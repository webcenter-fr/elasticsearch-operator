# Optimize Rolling Restart with Shard Allocation Control (ECK-inspired)

## Goal

Optimize the rolling restart strategy to:
1. Avoid unnecessary shard movement/rebalancing during rolling restart
2. Avoid replica shard replication to other pods during rolling restart
3. Ensure restarted pods recover their primary shards on the same node (with data on disk)
4. Use the Node Shutdown API for ES >= 7.15.2 (modern, recommended approach)
5. Add safety predicates before proceeding with rolling restart
6. Keep the StatefulSet-based rolling update approach (one StatefulSet at a time)
7. **Operator must connect via headless service DNS** (not ClusterIP) to work even when pods are not ready

## Current State Analysis

### Current Implementation (`statefulset_reconciler.go`)

**Before upgrade** (`StatefulsetPhaseUpgradeStarted`):
- Calls `esHandler.DisableRoutingRebalance()` → sets `cluster.routing.rebalance.enable=none` (**persistent**)

**After upgrade** (`StatefulsetPhaseUpgradeFinished`):
- Calls `esHandler.EnableRoutingRebalance()` → sets `cluster.routing.rebalance.enable=all` (**persistent**)

**Gating logic**: Only one StatefulSet upgraded at a time via `Diff()` phase machine.

### Problems with Current Approach

| Issue | Impact |
|-------|--------|
| Uses `rebalance.enable=none` | Disables ALL rebalancing; may prevent primary shard recovery |
| Uses **persistent** settings | If operator crashes, cluster stuck with rebalancing disabled forever |
| No `allocation.enable=primaries` | Replica shards are replicated to other nodes during restart (wasteful) |
| No Node Shutdown API | Missing modern ES feature for graceful node restarts |
| No safety predicates | No checks for cluster health, replica availability, shard conflicts |
| Operator uses ClusterIP service | Cannot connect when pods are not ready (service only routes to ready pods) |

### Critical Finding: Operator Connection via Headless Service

**Problem**: The operator connects via ClusterIP service (`GetGlobalServiceName`):
- Service only routes to **ready** pods (no `PublishNotReadyAddresses`)
- During rolling restart, pods become not-ready → operator may lose connection
- If operator cannot connect, it cannot re-enable rebalancing/allocation after upgrade

**Solution**: Use **headless service DNS names** for individual pods:
- Headless services have `PublishNotReadyAddresses: true` → resolve even for not-ready pods
- Pod DNS format: `<pod-name>.<headless-service>.<namespace>.svc:9200`
- Example: `myes-master-es-0.myes-master-headless-es.mynamespace.svc:9200`
- The elasticsearch v9 client supports multiple addresses → tries each until one works

**Existing helpers** (in `helper.go`):
- `GetNodeNames(es)` → all pod names across all node groups
- `GetNodeGroupNodeNames(es, nodeGroupName)` → pod names for a specific node group
- `GetNodeGroupServiceNameHeadless(es, nodeGroupName)` → headless service name

**Implementation** in `getElasticsearchHandler` (`elasticsearch_controller.go`):
```go
// Instead of single ClusterIP address:
addresses = append(addresses, fmt.Sprintf("http://%s.%s.svc:9200", serviceName, es.Namespace))

// Use all pod DNS names via headless services:
for _, nodeGroup := range es.Spec.NodeGroups {
    headlessService := GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)
    nodeNames := GetNodeGroupNodeNames(es, nodeGroup.Name)
    for _, nodeName := range nodeNames {
        addresses = append(addresses, fmt.Sprintf("%s://%s.%s.%s.svc:9200",
            scheme, nodeName, headlessService, es.Namespace))
    }
}
```

**Benefits**:
- Operator can always connect, even when pods are not ready
- No dependency on cluster health status (`waitClusterStatus`)
- Users can change `waitClusterStatus` without breaking the operator
- Multiple addresses provide failover

### Readiness Probe: No Changes Needed

The readiness probe stays as-is:
- First boot: waits for cluster health (`wait_for_status=${PROBE_WAIT_STATUS}`)
- Subsequent checks: just checks API reachable (starter file exists)
- Users can configure `waitClusterStatus` per node group (green/yellow/red)

**Why no changes needed**:
- The operator connects via headless service (not ClusterIP), so it doesn't depend on pod readiness
- The StatefulSet controller handles pod readiness independently
- With `allocation=primaries`, cluster may be "yellow" during restart — this is expected and acceptable
- Users who want stricter checks can set `waitClusterStatus: green`

## ECK Comparison (Reference Implementation)

ECK (`/tmp/eck/pkg/controller/elasticsearch/driver/stateful/upgrade.go`) uses:

### For ES >= 7.15.2: Node Shutdown API
```go
// Before deleting pod:
PUT /_nodes/{node_id}/shutdown {"type": "restart", "reason": "..."}
// Wait for shutdown status = COMPLETE
// Delete pod
// After pod returns:
DELETE /_nodes/{node_id}/shutdown
```
- No need to disable allocation/rebalancing
- Elasticsearch handles shard migration gracefully

### For ES < 7.15.2: Allocation Filtering (Legacy)
```go
// Before upgrade:
PUT /_cluster/settings {"transient": {"cluster.routing.allocation.enable": "primaries"}}
// Flush indices
// After upgrade:
PUT /_cluster/settings {"transient": {"cluster.routing.allocation.enable": "all"}}
```
- Uses **transient** settings (auto-reset on cluster restart)
- `primaries` allows primary allocation but blocks replica allocation

### ECK Safety Predicates
1. `only_restart_healthy_node_if_green_or_yellow` - Don't restart if cluster is RED
2. `require_started_replica` - Don't delete node with primary if no STARTED replica
3. `do_not_delete_pods_with_same_shards` - Don't delete 2 pods with same shards
4. `one_master_at_a_time` - Only one master-eligible node at a time
5. `do_not_delete_last_master_if_all_master_ineligible_nodes_are_not_upgraded`
6. `do_not_delete_all_members_of_a_tier`

## Proposed Solution

### Phase 1: Use Headless Service DNS for Operator Connection (Critical)

**Problem**: Operator uses ClusterIP service which only routes to ready pods.

**Solution**: Use headless service DNS names for all pods.

**File**: `internal/controller/elasticsearch/elasticsearch_controller.go`

**Function**: `getElasticsearchHandler` (around line 418)

**Change**:
```go
func (h *ElasticsearchReconciler) getElasticsearchHandler(ctx context.Context, es *elasticsearchcrd.Elasticsearch, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {
	addresses := []string{}

	// Get Elasticsearch credentials
	secret := &corev1.Secret{}
	if err = h.Client().Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: GetSecretNameForCredentials(es)}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("Secret %s not yet exist, try later", GetSecretNameForCredentials(es))
			return nil, nil
		}
		log.Errorf("Error when get resource: %s", err.Error())
		return nil, err
	}

	// Determine scheme
	scheme := "http"
	if es.Spec.Tls.IsTlsEnabled() {
		scheme = "https"
	}

	// Build addresses from headless service DNS names (all pods, ready or not)
	// This ensures operator can always connect, even during rolling restart
	for _, nodeGroup := range es.Spec.NodeGroups {
		headlessService := GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)
		nodeNames := GetNodeGroupNodeNames(es, nodeGroup.Name)
		for _, nodeName := range nodeNames {
			addresses = append(addresses, fmt.Sprintf("%s://%s.%s.%s.svc:9200",
				scheme, nodeName, headlessService, es.Namespace))
		}
	}

	// Fallback: also add the global ClusterIP service (for load balancing across ready pods)
	serviceName := GetGlobalServiceName(es)
	addresses = append(addresses, fmt.Sprintf("%s://%s.%s.svc:9200", scheme, serviceName, es.Namespace))

	cfg := &elasticsearch.Config{
		Addresses:         addresses,
		Username:          "elastic",
		Password:          string(secret.Data["elastic"]),
		TLSSkipVerify:     es.Spec.Tls.IsSelfManagedSecretForTls(),
		AllowInsecureHTTP: !es.Spec.Tls.IsTlsEnabled(),
		Timeout:           common.ESClientTimeout,
	}

	// ... rest of function (BYO TLS handling, etc.)
}
```

**Why this works**:
- Headless services have `PublishNotReadyAddresses: true` → DNS resolves even for not-ready pods
- The elasticsearch v9 client supports multiple addresses → tries each until one works
- Operator can always connect, even during rolling restart when pods are not ready
- No dependency on cluster health status
- Users can change `waitClusterStatus` without breaking the operator

**Edge cases**:
- If no pods exist yet (initial bootstrap), addresses list is empty → client creation fails gracefully → operator retries
- If all pods are down, client cannot connect → operator sets `Health=Unreachable` and retries (existing behavior)

### Phase 2: Add Transient Settings to es-handler

**Problem**: Current es-handler uses persistent settings (dangerous if operator crashes).

**Solution**: Add transient setting methods to es-handler.

**See**: Separate plan at `.opencode/plans/es-handler-rolling-restart-enhancements.md`

**Summary of methods to add**:
- `DisableReplicaShardsAllocationTransient()` → `transient.cluster.routing.allocation.enable=primaries`
- `EnableShardAllocationTransient()` → `transient.cluster.routing.allocation.enable=all`
- `DisableRoutingRebalanceTransient()` → `transient.cluster.routing.rebalance.enable=none`
- `EnableRoutingRebalanceTransient()` → `transient.cluster.routing.rebalance.enable=all`
- `RemoveTransientAllocationSettings()` → clear transient settings

### Phase 3: Add Node Shutdown API to es-handler

**Available in**: `github.com/disaster37/elasticsearch/v9` client → `client.Shutdown()`

**See**: Separate plan at `.opencode/plans/es-handler-rolling-restart-enhancements.md`

**Summary of methods to add**:
- `PutNodeShutdown(ctx, nodeID, type, reason)` → `PUT /_nodes/{node_id}/shutdown`
- `GetNodeShutdown(ctx, nodeID)` → `GET /_nodes/{node_id}/shutdown`
- `DeleteNodeShutdown(ctx, nodeID)` → `DELETE /_nodes/{node_id}/shutdown`
- `GetNodeNameToIDMap(ctx)` → map pod names to ES node IDs
- `GetVersion(ctx)` → get ES version for strategy selection

### Phase 4: Add Shard Listing to es-handler

**Available in**: `github.com/disaster37/elasticsearch/v9` client → `client.Cat()`

**See**: Separate plan at `.opencode/plans/es-handler-rolling-restart-enhancements.md`

**Summary of methods to add**:
- `GetShards(ctx)` → list all shards
- `GetShardsByNode(ctx)` → shards grouped by node
- `CountStartedReplicas(ctx, index, shard, excludeNode)` → for safety predicates

### Phase 5: Create Rolling Restart Orchestrator

**File**: `internal/controller/elasticsearch/rolling_restart.go` (new file)

**Purpose**: Encapsulate the rolling restart logic with version-aware strategy selection.

```go
package elasticsearch

import (
	"context"
	"fmt"
	"strings"

	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RollingRestartStrategy defines the strategy for rolling restart
type RollingRestartStrategy string

const (
	// StrategyNodeShutdown uses the Node Shutdown API (ES >= 7.15.2)
	StrategyNodeShutdown RollingRestartStrategy = "node-shutdown"
	// StrategyAllocationFilter uses allocation filtering (ES < 7.15.2)
	StrategyAllocationFilter RollingRestartStrategy = "allocation-filter"
)

// RollingRestartOrchestrator manages the rolling restart process
type RollingRestartOrchestrator struct {
	esHandler    eshandler.ElasticsearchHandler
	k8sClient    client.Client
	logger       *logrus.Entry
	strategy     RollingRestartStrategy
	esVersion    string
	nodeNameToID map[string]string // pod name -> ES node ID
}

// NewRollingRestartOrchestrator creates a new orchestrator
func NewRollingRestartOrchestrator(
	esHandler eshandler.ElasticsearchHandler,
	k8sClient client.Client,
	logger *logrus.Entry,
) (*RollingRestartOrchestrator, error) {
	ctx := context.Background()

	// Get ES version
	version, err := esHandler.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ES version: %w", err)
	}

	// Determine strategy based on version
	strategy := StrategyAllocationFilter
	if supportsNodeShutdown(version) {
		strategy = StrategyNodeShutdown
	}

	// Get node name to ID mapping
	nodeNameToID, err := esHandler.GetNodeNameToIDMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes info: %w", err)
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

// supportsNodeShutdown returns true if ES version supports Node Shutdown API
func supportsNodeShutdown(version string) bool {
	// Node Shutdown API available since ES 7.15.2
	// Parse version and compare
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major := parts[0]
	minor := parts[1]
	patch := strings.TrimSuffix(parts[2], "-SNAPSHOT")

	if major == "7" {
		if minor == "15" && patch >= "2" {
			return true
		}
		if minor > "15" {
			return true
		}
	}
	if major == "8" || major == "9" {
		return true
	}
	return false
}

// PrepareForRollingRestart prepares the cluster before starting rolling restart
func (r *RollingRestartOrchestrator) PrepareForRollingRestart(ctx context.Context) error {
	r.logger.Infof("Preparing for rolling restart using strategy: %s (ES version: %s)", r.strategy, r.esVersion)

	switch r.strategy {
	case StrategyNodeShutdown:
		// Node Shutdown API: no need to disable allocation/rebalancing
		// Shutdown requests will be sent per-pod before deletion
		r.logger.Info("Using Node Shutdown API strategy - no cluster settings to change")
		return nil

	case StrategyAllocationFilter:
		// Legacy strategy: disable replica allocation and rebalancing (transient)
		r.logger.Info("Disabling replica shard allocation (transient)")
		if err := r.esHandler.DisableReplicaShardsAllocationTransient(); err != nil {
			return fmt.Errorf("failed to disable replica allocation: %w", err)
		}

		r.logger.Info("Disabling routing rebalance (transient)")
		if err := r.esHandler.DisableRoutingRebalanceTransient(); err != nil {
			return fmt.Errorf("failed to disable rebalance: %w", err)
		}

		// Flush indices to optimize recovery
		r.logger.Info("Flushing indices")
		if err := r.esHandler.Flush(ctx); err != nil {
			r.logger.Warnf("Failed to flush indices: %s", err)
		}

		return nil
	}

	return nil
}

// CompleteRollingRestart cleans up after rolling restart is finished
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
			if s.Type == "restart" && s.Status == eshandler.ShutdownComplete {
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

// RequestNodeRestart requests a graceful restart for a specific pod (Node Shutdown API only)
func (r *RollingRestartOrchestrator) RequestNodeRestart(ctx context.Context, podName string) error {
	if r.strategy != StrategyNodeShutdown {
		return nil // No-op for allocation filter strategy
	}

	nodeID, ok := r.nodeNameToID[podName]
	if !ok {
		return fmt.Errorf("node ID not found for pod %s", podName)
	}

	r.logger.Infof("Requesting node shutdown for pod %s (node ID: %s)", podName, nodeID)
	return r.esHandler.PutNodeShutdown(ctx, nodeID, eshandler.ShutdownTypeRestart, "rolling-restart")
}

// WaitForNodeShutdown waits for a node shutdown to complete (Node Shutdown API only)
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

// ClearNodeShutdown clears a completed node shutdown (Node Shutdown API only)
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

// CheckPredicates checks safety predicates before proceeding with rolling restart
func (r *RollingRestartOrchestrator) CheckPredicates(ctx context.Context, stsList *appv1.StatefulSetList) (bool, string, error) {
	// Predicate 1: Cluster health must be green or yellow (not red)
	health, err := r.esHandler.ClusterHealth()
	if err != nil {
		return false, "", fmt.Errorf("failed to get cluster health: %w", err)
	}
	if health.Status == "red" {
		return false, "cluster health is RED", nil
	}

	// Predicate 2: Check that each pod being upgraded has started replicas
	// for its primaries (simplified version of ECK's require_started_replica)
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

	return true, "", nil
}
```

### Phase 6: Update StatefulSet Reconciler

**File**: `internal/controller/elasticsearch/statefulset_reconciler.go`

**Changes in `Diff()` method**:

Replace the current rebalance disable/enable logic with the orchestrator:

```go
// In Diff(), replace lines 395-405 (enable rebalance) and 422-430 (disable rebalance)

// When starting upgrade (StatefulsetPhaseUpgradeStarted):
if data["phase"] == StatefulsetPhaseUpgradeStarted {
	// Create orchestrator
	orchestrator, err := NewRollingRestartOrchestrator(esHandler, r.Client(), logger)
	if err != nil {
		return diff, res, errors.Wrap(err, "Error when create rolling restart orchestrator")
	}
	data["rollingRestartOrchestrator"] = orchestrator

	// Check predicates before starting
	stsList := &appv1.StatefulSetList{Items: currentStatefulsets}
	ok, reason, err := orchestrator.CheckPredicates(ctx, stsList)
	if err != nil {
		return diff, res, errors.Wrap(err, "Error when check predicates")
	}
	if !ok {
		logger.Warnf("Predicates not satisfied, delaying rolling restart: %s", reason)
		return diff, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Prepare cluster for rolling restart
	if err := orchestrator.PrepareForRollingRestart(ctx); err != nil {
		logger.Warnf("Error when prepare rolling restart: %s", err)
		// Don't block, but log the error
	}
}

// When finishing upgrade (StatefulsetPhaseUpgradeFinished):
if data["phase"] == StatefulsetPhaseUpgradeFinished {
	// Get orchestrator from data
	v, err := helper.Get(data, "rollingRestartOrchestrator")
	if err == nil {
		if orchestrator, ok := v.(*RollingRestartOrchestrator); ok {
			// Complete rolling restart (re-enable settings)
			if err := orchestrator.CompleteRollingRestart(ctx); err != nil {
				return diff, res, errors.Wrap(err, "Error when complete rolling restart")
			}
		}
	}
}
```

**Note**: The orchestrator is stored in `data` map to persist across reconcile loops within the same upgrade cycle.

### Phase 7: Update Tests

**Files to update**:
1. `internal/controller/elasticsearch/elasticsearch_controller_test.go` - Test headless service DNS addresses
2. Create `internal/controller/elasticsearch/rolling_restart_test.go` - Test orchestrator logic
3. Update es-handler tests in `/projects/es-handler/`

**Test cases**:
- Headless service DNS address construction
- Orchestrator strategy selection (ES version detection)
- Predicate checks (cluster health, replica availability)
- Transient settings set/clear
- Node Shutdown API calls (mocked)

## Implementation Order

1. **Phase 1**: Use headless service DNS for operator connection - **CRITICAL, do first**
2. **Phase 2**: Add transient settings to es-handler (see separate plan)
3. **Phase 3**: Add Node Shutdown API to es-handler (see separate plan)
4. **Phase 4**: Add shard listing to es-handler (see separate plan)
5. **Phase 5**: Create rolling restart orchestrator
6. **Phase 6**: Update statefulset reconciler to use orchestrator
7. **Phase 7**: Update tests

## Validation Plan

### Unit Tests
```bash
# es-handler tests
cd /projects/es-handler && go test ./...

# Operator tests
cd /projects/elasticsearch-operator && go test ./internal/controller/elasticsearch/...
```

### Integration Test Scenario

1. Create a 3-node ES cluster (1 master, 2 data)
2. Create an index with 1 primary + 1 replica
3. Trigger a rolling restart (change image tag or config)
4. Verify:
   - Operator connects via headless service DNS (check logs for addresses)
   - Transient settings are set before restart (`allocation=primaries`, `rebalance=none`)
   - Only one StatefulSet is upgraded at a time
   - Pods become ready (readiness probe passes)
   - No replica shards are replicated to other nodes during restart
   - Primary shards recover on original nodes
   - Transient settings are cleared after restart
   - Cluster health returns to green

### Manual Verification Commands
```bash
# Check operator can connect via headless service
kubectl exec -it <operator-pod> -- nslookup myes-master-es-0.myes-master-headless-es.mynamespace.svc

# Check transient settings during restart
kubectl exec -it <es-pod> -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_cluster/settings?include_defaults=true | jq '.transient'

# Check shard allocation during restart
kubectl exec -it <es-pod> -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_cat/shards?v

# Check node shutdown status (ES >= 7.15.2)
kubectl exec -it <es-pod> -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_nodes/shutdown
```

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Headless service DNS not resolving | Verify headless service exists and has `PublishNotReadyAddresses: true` |
| Too many addresses for large clusters | Limit to first N pods or use global headless service (future enhancement) |
| Transient settings lost on cluster restart | Acceptable - settings are meant to be temporary; cluster restarts to default |
| Node Shutdown API not available (ES < 7.15.2) | Fall back to allocation filter strategy |
| Operator crashes during restart | Transient settings auto-reset on cluster restart; no persistent damage |
| Predicates too strict, block restart | Log reason clearly; allow override via annotation (future enhancement) |
| User changes `waitClusterStatus` | No impact - operator uses headless service, not cluster status |

## Open Questions / Future Enhancements

1. **Global headless service**: Create a single global headless service instead of per-node-group DNS names (simpler for large clusters)
2. **Annotation to disable predicates**: Add `elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates` annotation (like ECK)
3. **MaxUnavailable budget**: Add `maxUnavailable` support to control how many pods can be down
4. **Pre-stop hook**: Add pre-stop hook to pods for graceful shutdown (like ECK's lifecycle hook)
5. **Allocation delay**: Support `allocation_delay` parameter in Node Shutdown API
6. **Force upgrade**: Add annotation to force upgrade even if predicates fail

## Files to Modify

### es-handler (separate repo at `/projects/es-handler`)
- `cluster.go` - Add transient settings methods
- `elasticsearch.go` - Update interface
- `mocks/elasticsearch_handler.go` - Regenerate mocks
- `shutdown.go` (new) - Add Node Shutdown API methods
- `shard.go` (new) - Add shard listing methods
- `nodes.go` (new) - Add node info methods

### elasticsearch-operator
- `internal/controller/elasticsearch/elasticsearch_controller.go` - Use headless service DNS for operator connection
- `internal/controller/elasticsearch/statefulset_reconciler.go` - Use orchestrator
- `internal/controller/elasticsearch/rolling_restart.go` (new) - Orchestrator
- `internal/controller/elasticsearch/elasticsearch_controller_test.go` - Update tests
- `internal/controller/elasticsearch/rolling_restart_test.go` (new) - Orchestrator tests

## Dependencies

- `github.com/disaster37/es-handler/v9` - Needs new methods (Phases 2-4, see separate plan)
- `github.com/disaster37/elasticsearch/v9` - Already has Shutdown, Nodes, Cat services

## Related Plans

- **es-handler enhancements**: `.opencode/plans/es-handler-rolling-restart-enhancements.md`
