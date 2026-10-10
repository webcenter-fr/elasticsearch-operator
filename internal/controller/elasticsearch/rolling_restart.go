package elasticsearch

// Rolling restart orchestration inspired by ECK (Elastic Cloud on Kubernetes)
// https://github.com/elastic/cloud-on-k8s
// Adapted for our operator's architecture and Apache 2.0 license.
// No ECK code has been copied directly.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RollingRestartStrategy defines the strategy used to prepare the cluster
// before a rolling restart.
type RollingRestartStrategy string

const (
	// StrategyNodeShutdown uses the Elasticsearch Node Shutdown API (ES >= 7.15.2).
	// Elasticsearch migrates shards away from the node gracefully before the
	// pod is deleted, so no cluster settings need to be changed.
	StrategyNodeShutdown RollingRestartStrategy = "node-shutdown"
	// StrategyAllocationFilter uses transient allocation settings (ES < 7.15.2).
	// Replica shard allocation and shard rebalancing are disabled (transient,
	// auto-reset on cluster restart) while pods are restarted one by one.
	StrategyAllocationFilter RollingRestartStrategy = "allocation-filter"
)

// RollingRestartOrchestrator manages the rolling restart process with a
// version-aware strategy selection.
type RollingRestartOrchestrator struct {
	esHandler    eshandler.ElasticsearchHandler
	k8sClient    client.Client
	logger       *logrus.Entry
	strategy     RollingRestartStrategy
	esVersion    string
	nodeNameToID map[string]string // pod name -> ES node ID
}

// NewRollingRestartOrchestrator creates a new orchestrator. It detects the
// Elasticsearch version to select the rolling restart strategy and builds the
// pod name to node ID mapping used by the Node Shutdown API.
func NewRollingRestartOrchestrator(
	esHandler eshandler.ElasticsearchHandler,
	k8sClient client.Client,
	logger *logrus.Entry,
) (*RollingRestartOrchestrator, error) {
	ctx := context.Background()

	version, err := esHandler.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Elasticsearch version: %w", err)
	}

	strategy := StrategyAllocationFilter
	if supportsNodeShutdown(version) {
		strategy = StrategyNodeShutdown
	}

	nodeNameToID, err := esHandler.GetNodeNameToIDMap(ctx)
	if err != nil {
		// The node name -> ID map is only used to request per-node shutdowns
		// (Node Shutdown API). It must not block the rolling restart lifecycle
		// (especially the completion that re-enables shard allocation and
		// rebalancing), so log and continue with an empty map.
		logger.Warnf("Failed to get Elasticsearch node name to ID map, node shutdown requests will fail until it succeeds: %s", err)
		nodeNameToID = map[string]string{}
	}

	return &RollingRestartOrchestrator{
		esHandler:    esHandler,
		k8sClient:    k8sClient,
		logger:       logger,
		strategy:     strategy,
		esVersion:    version,
		nodeNameToID: nodeNameToID,
	}, nil
}

// Strategy returns the rolling restart strategy selected for the cluster.
func (r *RollingRestartOrchestrator) Strategy() RollingRestartStrategy {
	return r.strategy
}

// ESVersion returns the detected Elasticsearch version.
func (r *RollingRestartOrchestrator) ESVersion() string {
	return r.esVersion
}

// supportsNodeShutdown returns true if the Elasticsearch version supports the
// Node Shutdown API (available since ES 7.15.2).
func supportsNodeShutdown(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	patch := 0
	if len(parts) > 2 {
		patchStr := parts[2]
		// Strip any qualifier suffix (e.g. "-SNAPSHOT", "-alpha1")
		if i := strings.IndexAny(patchStr, "-+"); i >= 0 {
			patchStr = patchStr[:i]
		}
		patch, _ = strconv.Atoi(patchStr)
	}

	switch {
	case major >= 8:
		return true
	case major == 7:
		return minor > 15 || (minor == 15 && patch >= 2)
	}

	return false
}

// PrepareForRollingRestart prepares the cluster before starting a rolling restart.
func (r *RollingRestartOrchestrator) PrepareForRollingRestart(ctx context.Context) error {
	r.logger.Infof("Preparing for rolling restart using strategy: %s (ES version: %s)", r.strategy, r.esVersion)

	switch r.strategy {
	case StrategyNodeShutdown:
		// Node Shutdown API: no need to disable allocation/rebalancing.
		// Shutdown requests are sent per-pod before deletion.
		r.logger.Info("Using Node Shutdown API strategy - no cluster settings to change")
		return nil

	case StrategyAllocationFilter:
		// Legacy strategy: disable replica allocation and rebalancing (transient).
		r.logger.Info("Set transient cluster.routing.allocation.enable=primaries (replica allocation disabled)")
		if err := r.esHandler.DisableReplicaShardsAllocationTransient(); err != nil {
			return fmt.Errorf("failed to disable replica allocation: %w", err)
		}

		r.logger.Info("Set transient cluster.routing.rebalance.enable=none (shard rebalancing disabled)")
		if err := r.esHandler.DisableRoutingRebalanceTransient(); err != nil {
			return fmt.Errorf("failed to disable rebalance: %w", err)
		}

		// Flush indices to optimize recovery
		r.logger.Info("Flushing indices for faster recovery")
		if err := r.esHandler.Flush(ctx); err != nil {
			r.logger.Warnf("Failed to flush indices (non-blocking): %s", err)
		}

		return nil
	}

	return nil
}

// CompleteRollingRestart cleans up after the rolling restart is finished.
func (r *RollingRestartOrchestrator) CompleteRollingRestart(ctx context.Context) error {
	r.logger.Infof("Completing rolling restart (strategy: %s)", r.strategy)

	switch r.strategy {
	case StrategyNodeShutdown:
		// Clean up any remaining shutdown requests
		r.logger.Info("Cleaning up node shutdown requests")
		shutdowns, err := r.esHandler.GetAllNodeShutdowns(ctx)
		if err != nil {
			return err
		}
		for _, s := range shutdowns {
			if s.Type == string(eshandler.ShutdownTypeRestart) && s.Status == eshandler.ShutdownComplete {
				r.logger.Infof("Cleaning up completed shutdown for node %s", s.NodeID)
				if err := r.esHandler.DeleteNodeShutdown(ctx, s.NodeID); err != nil {
					r.logger.Warnf("Failed to delete shutdown for node %s: %s", s.NodeID, err)
				}
			}
		}
		return nil

	case StrategyAllocationFilter:
		// Re-enable allocation and rebalancing (transient)
		r.logger.Info("Re-enabling shard allocation (transient)")
		if err := r.esHandler.EnableShardAllocationTransient(); err != nil {
			return fmt.Errorf("failed to enable shard allocation: %w", err)
		}

		r.logger.Info("Re-enabling routing rebalance (transient)")
		if err := r.esHandler.EnableRoutingRebalanceTransient(); err != nil {
			return fmt.Errorf("failed to enable rebalance: %w", err)
		}

		return nil
	}

	return nil
}

// RequestNodeShutdown requests a graceful shutdown for a pod (Node Shutdown API only).
// It is a no-op for the allocation filter strategy.
func (r *RollingRestartOrchestrator) RequestNodeShutdown(ctx context.Context, podName string) error {
	if r.strategy != StrategyNodeShutdown {
		return nil // No-op for allocation filter strategy
	}

	nodeID, ok := r.nodeNameToID[podName]
	if !ok {
		return fmt.Errorf("node ID not found for pod %s", podName)
	}

	r.logger.Infof("Requesting graceful shutdown for pod %s (node ID: %s)", podName, nodeID)
	return r.esHandler.PutNodeShutdown(ctx, nodeID, eshandler.ShutdownTypeRestart, "rolling-restart")
}

// RequestNodeRestart requests a graceful restart for a specific pod (Node Shutdown API only).
// It is an alias of RequestNodeShutdown.
func (r *RollingRestartOrchestrator) RequestNodeRestart(ctx context.Context, podName string) error {
	return r.RequestNodeShutdown(ctx, podName)
}

// WaitForNodeShutdown checks if a node shutdown is complete (Node Shutdown API only).
// It returns true immediately for the allocation filter strategy.
func (r *RollingRestartOrchestrator) WaitForNodeShutdown(ctx context.Context, podName string) (bool, error) {
	if r.strategy != StrategyNodeShutdown {
		return true, nil // No-op for allocation filter strategy
	}

	nodeID, ok := r.nodeNameToID[podName]
	if !ok {
		return false, fmt.Errorf("node ID not found for pod %s", podName)
	}

	info, err := r.esHandler.GetNodeShutdown(ctx, nodeID)
	if err != nil {
		return false, err
	}
	if info == nil {
		// No shutdown in progress, consider it complete
		return true, nil
	}

	r.logger.Debugf("Node shutdown status for %s: %s", podName, info.Status)
	return info.Status == eshandler.ShutdownComplete, nil
}

// IsNodeShutdownComplete checks if a node shutdown is complete (Node Shutdown API only).
// It is an alias of WaitForNodeShutdown.
func (r *RollingRestartOrchestrator) IsNodeShutdownComplete(ctx context.Context, podName string) (bool, error) {
	return r.WaitForNodeShutdown(ctx, podName)
}

// ClearNodeShutdown clears a completed node shutdown (Node Shutdown API only).
// It is a no-op for the allocation filter strategy.
func (r *RollingRestartOrchestrator) ClearNodeShutdown(ctx context.Context, podName string) error {
	if r.strategy != StrategyNodeShutdown {
		return nil // No-op for allocation filter strategy
	}

	nodeID, ok := r.nodeNameToID[podName]
	if !ok {
		// Node might have been removed, ignore
		return nil
	}

	r.logger.Infof("Clearing node shutdown for pod %s (node ID: %s)", podName, nodeID)
	return r.esHandler.DeleteNodeShutdown(ctx, nodeID)
}

// CheckPredicates checks cluster-level safety predicates before proceeding
// with a rolling restart of the given StatefulSets: the cluster health must
// not be RED, and each pod hosting a primary shard must have at least one
// STARTED replica elsewhere (when replicas are configured).
// Each predicate can be disabled with the
// elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates annotation.
func (r *RollingRestartOrchestrator) CheckPredicates(ctx context.Context, es *elasticsearchcrd.Elasticsearch, stsList *appv1.StatefulSetList) (bool, string, error) {
	// Predicate 1: Cluster health must be green or yellow (not red)
	if !IsPredicateDisabled(es, PredicateClusterHealthNotRed) {
		health, err := r.esHandler.ClusterHealth()
		if err != nil {
			return false, "", fmt.Errorf("failed to get cluster health: %w", err)
		}
		if health == nil {
			return false, "", fmt.Errorf("cluster health response is nil")
		}
		if health.Status == "red" {
			return false, "cluster health is RED", nil
		}
	}

	// Predicate 2: Check that each pod being upgraded has started replicas
	// for its primaries (simplified version of ECK's require_started_replica)
	if !IsPredicateDisabled(es, PredicateRequireStartedReplica) {
		shardsByNode, err := r.esHandler.GetShardsByNode(ctx)
		if err != nil {
			return false, "", fmt.Errorf("failed to get shards: %w", err)
		}

		for _, sts := range stsList.Items {
			// Get pods for this StatefulSet
			podList := &corev1.PodList{}
			if err := r.k8sClient.List(ctx, podList, client.InNamespace(sts.Namespace), client.MatchingLabels(sts.Spec.Selector.MatchLabels)); err != nil {
				return false, "", err
			}

			for _, pod := range podList.Items {
				shards := shardsByNode[pod.Name]
				for _, shard := range shards {
					if !shard.Primary {
						continue
					}
					// Check if there's at least one STARTED replica for this primary
					_, started, err := r.esHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, pod.Name)
					if err != nil {
						return false, "", err
					}
					// If replicas configured but none STARTED, block the restart
					if started == 0 {
						// Check if this shard has any replicas at all
						total, _, err := r.esHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, "")
						if err != nil {
							return false, "", err
						}
						if total > 0 {
							return false, fmt.Sprintf("no STARTED replica for primary shard %s-%s on pod %s", shard.Index, shard.Shard, pod.Name), nil
						}
					}
				}
			}
		}
	}

	return true, "", nil
}
