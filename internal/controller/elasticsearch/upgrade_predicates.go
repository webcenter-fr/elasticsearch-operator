package elasticsearch

// Upgrade safety predicates inspired by ECK (Elastic Cloud on Kubernetes)
// https://github.com/elastic/cloud-on-k8s
// Adapted for our operator's architecture and Apache 2.0 license.
// No ECK code has been copied directly.

import (
	"context"
	"fmt"
	"strings"

	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
)

// UpgradePredicate checks if it's safe to upgrade a pod during a rolling upgrade.
// It returns true when the pod can be upgraded, false with a human-readable
// reason when it must be delayed, or an error when the check itself failed.
type UpgradePredicate struct {
	Name  string
	Check func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error)
}

// UpgradeState holds the context needed to evaluate upgrade predicates.
type UpgradeState struct {
	ES              *elasticsearchcrd.Elasticsearch
	ESHandler       eshandler.ElasticsearchHandler
	HealthyPods     map[string]corev1.Pod // pod name -> pod (ready and not terminating)
	PodsToUpgrade   []corev1.Pod
	CurrentPods     []corev1.Pod
	ExpectedMasters []string
	DeletedPods     []corev1.Pod // pods currently terminating
	Logger          *logrus.Entry
}

// Predicate names. They are the values accepted by the
// elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates annotation.
const (
	PredicateClusterHealthNotRed   = "cluster_health_not_red"
	PredicateRequireStartedReplica = "require_started_replica_for_primary"
	PredicateOneMasterAtATime      = "one_master_at_a_time"
	PredicateSkipTerminatingPods   = "skip_terminating_pods"
)

// UpgradePredicates is the ordered list of safety predicates evaluated before
// upgrading a pod. Predicates can be disabled individually (or all at once)
// with the elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates annotation.
var UpgradePredicates = []UpgradePredicate{
	{
		Name: PredicateClusterHealthNotRed,
		Check: func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
			health, err := state.ESHandler.ClusterHealth()
			if err != nil {
				return false, "", err
			}
			if health == nil {
				return false, "", fmt.Errorf("cluster health response is nil")
			}
			if health.Status == "red" {
				return false, "cluster health is RED", nil
			}
			return true, "", nil
		},
	},
	{
		Name: PredicateRequireStartedReplica,
		Check: func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
			// Don't upgrade a node hosting a primary shard if no STARTED
			// replica exists elsewhere (when replicas are configured).
			shardsByNode, err := state.ESHandler.GetShardsByNode(ctx)
			if err != nil {
				return false, "", err
			}
			for _, shard := range shardsByNode[pod.Name] {
				if !shard.Primary {
					continue
				}
				_, started, err := state.ESHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, pod.Name)
				if err != nil {
					return false, "", err
				}
				if started == 0 {
					total, _, err := state.ESHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, "")
					if err != nil {
						return false, "", err
					}
					if total > 0 {
						return false, fmt.Sprintf("no STARTED replica for primary shard %s/%s", shard.Index, shard.Shard), nil
					}
				}
			}
			return true, "", nil
		},
	},
	{
		Name: PredicateOneMasterAtATime,
		Check: func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
			// Only restart one master-eligible node at a time
			if !IsMasterRole(state.ES, GetNodeGroupNameFromPodName(state.ES, pod.Name)) {
				return true, "", nil
			}
			for _, deleted := range state.DeletedPods {
				if IsMasterRole(state.ES, GetNodeGroupNameFromPodName(state.ES, deleted.Name)) {
					return false, "another master node is already being upgraded", nil
				}
			}
			return true, "", nil
		},
	},
	{
		Name: PredicateSkipTerminatingPods,
		Check: func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
			if pod.DeletionTimestamp != nil {
				return false, "pod is already terminating", nil
			}
			return true, "", nil
		},
	},
}

// IsPredicateDisabled returns true when the predicate is disabled via the
// elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates annotation.
// The annotation value is a comma-separated list of predicate names, or "*"
// to disable all predicates.
func IsPredicateDisabled(es *elasticsearchcrd.Elasticsearch, predicateName string) bool {
	if es.Annotations == nil {
		return false
	}
	val, ok := es.Annotations[elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation]
	if !ok {
		return false
	}
	for _, name := range strings.Split(val, ",") {
		name = strings.TrimSpace(name)
		if name == "*" || name == predicateName {
			return true
		}
	}
	return false
}

// ApplyPredicates runs all enabled predicates on a candidate pod.
// It returns true when the pod passed all predicates, false with a
// human-readable reason when a predicate blocked it, or an error when a
// predicate check itself failed.
func ApplyPredicates(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
	for _, predicate := range UpgradePredicates {
		if IsPredicateDisabled(state.ES, predicate.Name) {
			state.Logger.Infof("Predicate '%s' is disabled by annotation %s, skipping", predicate.Name, elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation)
			continue
		}
		ok, reason, err := predicate.Check(ctx, pod, state)
		if err != nil {
			return false, "", fmt.Errorf("predicate %s error: %w", predicate.Name, err)
		}
		if !ok {
			state.Logger.Infof("Pod %s blocked by predicate '%s': %s", pod.Name, predicate.Name, reason)
			return false, fmt.Sprintf("%s: %s", predicate.Name, reason), nil
		}
	}
	return true, "", nil
}
