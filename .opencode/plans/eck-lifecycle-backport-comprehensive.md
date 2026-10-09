# ECK Lifecycle Management Backport - Comprehensive Plan

## Executive Summary

After analyzing ECK (Elastic Cloud on Kubernetes) at `/tmp/eck`, I found that **ECK DOES use StatefulSets**, not direct pods. The package is literally named `stateful` (`/tmp/eck/pkg/controller/elasticsearch/driver/stateful/`). The key differences are in:

1. **How the operator connects to Elasticsearch** (smart URL provider vs ClusterIP only)
2. **Rolling upgrade orchestration** (Node Shutdown API + safety predicates)
3. **Pod suspension/downtime feature** (annotation-based suspend)
4. **Better logging and observability**

**CRITICAL**: Both ECK and our operator use `"elasticsearch-data"` as the VolumeClaimTemplate name, so **existing PVCs will be preserved** during any migration.

---

## Key Findings from ECK Analysis

### 1. ECK Uses StatefulSets (Not Direct Pods)

**ECK Structure:**
```
/tmp/eck/pkg/controller/elasticsearch/driver/stateful/
├── driver.go           # Main stateful driver
├── upgrade.go          # Rolling upgrade logic
├── upgrade_predicates.go  # Safety predicates (7 predicates)
├── upgrade_pods_deletion.go  # Pod deletion with ordering
├── node_shutdown.go    # Node Shutdown API wrapper
├── suspend.go          # Pod suspension feature
├── upscale.go          # Scale up logic
├── downscale.go        # Scale down logic
└── pvc_gc.go           # PVC garbage collection
```

**ECK StatefulSet Configuration** (`/tmp/eck/pkg/controller/elasticsearch/nodespec/statefulset.go`):
- Uses `OnDeleteStatefulSetStrategyType` (operator controls pod deletion, not K8s)
- Uses `ParallelPodManagement` (all pods created in parallel)
- Headless service per StatefulSet with `PublishNotReadyAddresses: true`
- VolumeClaimTemplate name: `"elasticsearch-data"` (same as ours!)

### 2. ECK's Smart URL Provider (Critical Improvement)

**Current Operator** (`elasticsearch_controller.go:418-472`):
```go
// ONLY uses ClusterIP global service - only routes to READY pods!
serviceName := GetGlobalServiceName(es)
addresses = append(addresses, fmt.Sprintf("http://%s.%s.svc:9200", serviceName, es.Namespace))
```

**ECK's URL Provider** (`/tmp/eck/pkg/controller/elasticsearch/services/services.go:188-257`):
```go
// Smart provider that tries multiple strategies:
// 1. Connect directly to READY pods via pod DNS (pod-name.sset-name.namespace:9200)
// 2. Fall back to RUNNING pods if no ready pods
// 3. Fall back to internal service URL as last resort
func (u *urlProvider) URL() (string, error) {
    pods := u.pods()
    // Prefer ready pods
    for _, p := range pods {
        if k8s.IsPodReady(p) { ready = append(ready, p) }
        if k8s.IsPodRunning(p) { running = append(running, p) }
    }
    switch {
    case len(ready) > 0: return randomESPodURL(ready)      // Direct pod DNS
    case len(running) > 0: return randomESPodURL(running)  // Running but not ready
    default: return u.svcURL                               // Service fallback
    }
}

// Pod DNS format: scheme://pod-name.statefulset-name.namespace:9200
func ElasticsearchPodURL(pod corev1.Pod) string {
    return fmt.Sprintf("%s://%s.%s.%s:%d", scheme, pod.Name, sset, pod.Namespace, 9200)
}
```

**Why This Matters:**
- Our operator connects via ClusterIP service → only routes to READY pods
- During rolling restart, pods become not-ready → operator loses connection
- ECK connects directly to pods → works even when pods are not ready
- This is **CRITICAL** for the rolling restart to work properly

### 3. ECK's Rolling Upgrade Strategy

**ECK's Approach** (`/tmp/eck/pkg/controller/elasticsearch/driver/stateful/upgrade.go`):

**For ES >= 7.15.2: Node Shutdown API**
```go
// 1. Request node shutdown (graceful)
PUT /_nodes/{node_id}/shutdown {"type": "restart", "reason": "..."}

// 2. Wait for shutdown to complete (shards migrated)
GET /_nodes/{node_id}/shutdown → status == COMPLETE

// 3. Delete the pod
DELETE pod (with UID precondition for safety)

// 4. After pod returns, clear shutdown
DELETE /_nodes/{node_id}/shutdown
```

**For ES < 7.15.2: Transient Allocation Settings**
```go
// Before upgrade (transient - auto-reset on cluster restart!)
PUT /_cluster/settings {"transient": {"cluster.routing.allocation.enable": "primaries"}}

// Flush indices for faster recovery
POST /_flush

// After upgrade
PUT /_cluster/settings {"transient": {"cluster.routing.allocation.enable": "all"}}
```

**Current Operator's Approach** (`statefulset_reconciler.go:395-434`):
```go
// Uses PERSISTENT settings (dangerous if operator crashes!)
esHandler.DisableRoutingRebalance()  // persistent: rebalance=none
// ... upgrade ...
esHandler.EnableRoutingRebalance()   // persistent: rebalance=all
```

**Problems with Current Approach:**
1. Uses **persistent** settings → if operator crashes, cluster stuck with rebalancing disabled
2. Only disables rebalance, doesn't disable replica allocation (`allocation.enable=primaries`)
3. No Node Shutdown API support
4. No safety predicates
5. Relies on StatefulSet controller for rolling update (less control)

### 4. ECK's Safety Predicates (7 Predicates)

From `/tmp/eck/pkg/controller/elasticsearch/driver/stateful/upgrade_predicates.go`:

| Predicate | Purpose |
|-----------|---------|
| `data_tier_with_higher_priority_must_be_upgraded_first` | Upgrade hot tier before warm/cold |
| `do_not_restart_healthy_node_if_MaxUnavailable_reached` | Respect maxUnavailable budget |
| `skip_already_terminating_pods` | Don't delete pods already terminating |
| `only_restart_healthy_node_if_green_or_yellow` | Don't restart if cluster is RED |
| `if_yellow_only_restart_upgrading_nodes_with_unassigned_replicas` | Yellow cluster: only restart upgrading nodes |
| `require_started_replica` | Don't delete node with primary if no STARTED replica |
| `one_master_at_a_time` | Only one master-eligible node at a time |
| `do_not_delete_last_master_if_all_master_ineligible_nodes_are_not_upgraded` | Don't delete last master until data nodes upgraded |
| `do_not_delete_pods_with_same_shards` | Don't delete 2 pods with same shards |
| `do_not_delete_all_members_of_a_tier` | Don't delete all pods of a tier |

### 5. ECK's Pod Suspension Feature (Downtime)

**ECK Annotation:** `eck.k8s.elastic.co/suspend` (comma-separated pod names)

**How It Works** (`/tmp/eck/pkg/controller/elasticsearch/driver/stateful/suspend.go` + `/tmp/eck/pkg/controller/elasticsearch/initcontainer/suspend.go`):

1. User annotates Elasticsearch CR: `eck.k8s.elastic.co/suspend: "pod-1,pod-2"`
2. Operator creates a ConfigMap with suspended pod names
3. Each pod has an init container that checks if its name is in the file
4. If suspended, init container loops (`sleep 10`) preventing main container from starting
5. Operator deletes running pods that should be suspended (so they restart and get suspended)
6. To resume: remove pod name from annotation → init container exits → ES starts

**Init Container Script:**
```bash
#!/usr/bin/env bash
while [[ $(grep -Exc $HOSTNAME /mnt/elastic-internal/scripts/suspended_pods.txt) -eq 1 ]]; do
  echo "Pod suspended via eck.k8s.elastic.co/suspend annotation"
  sleep 10
done
```

**Key Feature for User's Request:**
- Stop a pod WITHOUT recreating it (for PVC access when crashback)
- The pod stays in "Init" state, PVC remains mounted
- Can exec into the pod to access data
- Remove annotation to resume normal operation

### 6. PVC Naming Compatibility (CRITICAL)

**ECK Volume Claim Template Name:** `"elasticsearch-data"` (`/tmp/eck/pkg/controller/elasticsearch/volume/names.go:30`)

**Our Operator Volume Claim Template Name:** `"elasticsearch-data"` (`statefulset_builder.go:878`)

**Result:** ✅ **PVC names are IDENTICAL** - existing data will be preserved!

PVC naming pattern in StatefulSets:
- `<volumeClaimTemplateName>-<statefulSetName>-<ordinal>`
- Example: `elasticsearch-data-my-es-master-0`

Since both use `elasticsearch-data` as the VCT name, **no data migration needed**.

---

## Copyright & Licensing Considerations

**ECK License:** Elastic License 2.0 (source-available, not OSI-approved open source)
**Our Operator License:** Apache License 2.0

**Approach:**
- ✅ **DO:** Learn from ECK's architecture and adapt concepts to our codebase
- ✅ **DO:** Write original code that implements similar functionality
- ✅ **DO:** Reference ECK's approach in comments (e.g., "Inspired by ECK's approach...")
- ❌ **DON'T:** Copy ECK code directly (license incompatibility)
- ❌ **DON'T:** Copy ECK's file structure or naming conventions verbatim

**Attribution:** Add a note in our code/docs acknowledging ECK's influence:
```go
// Rolling restart orchestration inspired by ECK (Elastic Cloud on Kubernetes)
// https://github.com/elastic/cloud-on-k8s
// Adapted for our operator's architecture and Apache 2.0 license.
```

---

## Implementation Plan

### Phase 1: ES-Handler Enhancements (Prerequisite)

**Repository:** `/projects/es-handler`
**Reference Plan:** `.opencode/plans/es-handler-rolling-restart-enhancements.md`

Add these methods to `ElasticsearchHandler` interface:

**File: `cluster.go`** (modify existing)
```go
// Transient settings (safer than persistent)
DisableReplicaShardsAllocationTransient() error  // transient: allocation=primaries
EnableShardAllocationTransient() error           // transient: allocation=all
DisableRoutingRebalanceTransient() error         // transient: rebalance=none
EnableRoutingRebalanceTransient() error          // transient: rebalance=all
RemoveTransientAllocationSettings() error        // clear transient settings

// Cluster info
GetVersion(ctx context.Context) (string, error)

// Flush
Flush(ctx context.Context) error
```

**File: `shutdown.go`** (new)
```go
type ShutdownType string
type ShutdownStatus string
type NodeShutdownInfo struct { ... }

PutNodeShutdown(ctx, nodeID, type, reason) error
GetNodeShutdown(ctx, nodeID) (*NodeShutdownInfo, error)
GetAllNodeShutdowns(ctx) ([]NodeShutdownInfo, error)
DeleteNodeShutdown(ctx, nodeID) error
```

**File: `nodes.go`** (new)
```go
type NodeInfo struct { ID, Name, Version string; Roles []string; ... }

GetNodesInfo(ctx) (map[string]NodeInfo, error)
GetNodeNameToIDMap(ctx) (map[string]string, error)
IsNodeInCluster(ctx, nodeName) (bool, error)
```

**File: `shard.go`** (new)
```go
type ShardInfo struct { Index, Shard string; Primary bool; State, NodeName string; ... }

GetShards(ctx) ([]ShardInfo, error)
GetShardsByNode(ctx) (map[string][]ShardInfo, error)
CountStartedReplicas(ctx, index, shard, excludeNode) (total, started int, err error)
```

**Update:** `elasticsearch.go` (interface), `mocks/elasticsearch_handler.go` (regenerate)

**Tests:** Add unit tests for all new methods

---

### Phase 2: Smart ES Client Connection (Critical)

**File:** `internal/controller/elasticsearch/elasticsearch_controller.go`
**Function:** `getElasticsearchHandler` (line 418)

**Current:** Only uses ClusterIP global service (ready pods only)

**New Approach (inspired by ECK's URL provider):**

```go
func (h *ElasticsearchReconciler) getElasticsearchHandler(ctx context.Context, es *elasticsearchcrd.Elasticsearch, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {
    // ... get credentials as before ...

    addresses := []string{}
    scheme := "http"
    if es.Spec.Tls.IsTlsEnabled() {
        scheme = "https"
    }

    // Strategy 1: Add headless service DNS for each pod (ready or not)
    // This is CRITICAL - allows connection during rolling restart
    for _, nodeGroup := range es.Spec.NodeGroups {
        headlessService := GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)
        nodeNames := GetNodeGroupNodeNames(es, nodeGroup.Name)
        for _, nodeName := range nodeNames {
            addresses = append(addresses, fmt.Sprintf("%s://%s.%s.%s.svc:9200",
                scheme, nodeName, headlessService, es.Namespace))
        }
    }

    // Strategy 2: Add global ClusterIP service as fallback (load balancing)
    serviceName := GetGlobalServiceName(es)
    addresses = append(addresses, fmt.Sprintf("%s://%s.%s.svc:9200",
        scheme, serviceName, es.Namespace))

    log.Infof("Elasticsearch client configured with %d addresses (headless pod DNS + global service)", len(addresses))

    // ... rest of function (TLS handling, client creation) ...
}
```

**Why This Works:**
- Headless services have `PublishNotReadyAddresses: true` (already set in `service_builder.go:111`)
- DNS resolves even for not-ready pods: `pod-name.headless-service.namespace.svc`
- ES client tries multiple addresses → failover works
- Operator can always connect, even during rolling restart

**Logging:** Add INFO level log showing all addresses being used

---

### Phase 3: Rolling Restart Orchestrator

**File:** `internal/controller/elasticsearch/rolling_restart.go` (new)

**Purpose:** Encapsulate rolling restart logic with version-aware strategy selection.

**Key Components:**

```go
// Strategy selection based on ES version
type RollingRestartStrategy string

const (
    StrategyNodeShutdown     RollingRestartStrategy = "node-shutdown"      // ES >= 7.15.2
    StrategyAllocationFilter RollingRestartStrategy = "allocation-filter"  // ES < 7.15.2
)

type RollingRestartOrchestrator struct {
    esHandler    eshandler.ElasticsearchHandler
    k8sClient    client.Client
    logger       *logrus.Entry
    strategy     RollingRestartStrategy
    esVersion    string
    nodeNameToID map[string]string  // pod name -> ES node ID
}

// Prepare cluster before rolling restart
func (r *RollingRestartOrchestrator) PrepareForRollingRestart(ctx context.Context) error {
    r.logger.Infof("Preparing rolling restart with strategy: %s (ES version: %s)", r.strategy, r.esVersion)

    switch r.strategy {
    case StrategyNodeShutdown:
        r.logger.Info("Using Node Shutdown API - no cluster settings to change")
        return nil
    case StrategyAllocationFilter:
        r.logger.Info("Setting transient allocation=primaries (replica allocation disabled)")
        if err := r.esHandler.DisableReplicaShardsAllocationTransient(); err != nil {
            return fmt.Errorf("failed to disable replica allocation: %w", err)
        }
        r.logger.Info("Setting transient rebalance=none (shard rebalancing disabled)")
        if err := r.esHandler.DisableRoutingRebalanceTransient(); err != nil {
            return fmt.Errorf("failed to disable rebalance: %w", err)
        }
        r.logger.Info("Flushing indices for faster recovery")
        if err := r.esHandler.Flush(ctx); err != nil {
            r.logger.Warnf("Failed to flush indices (non-blocking): %s", err)
        }
        return nil
    }
    return nil
}

// Complete rolling restart (cleanup)
func (r *RollingRestartOrchestrator) CompleteRollingRestart(ctx context.Context) error {
    r.logger.Infof("Completing rolling restart (strategy: %s)", r.strategy)

    switch r.strategy {
    case StrategyNodeShutdown:
        // Clean up completed shutdowns
        shutdowns, err := r.esHandler.GetAllNodeShutdowns(ctx)
        if err != nil {
            return err
        }
        for _, s := range shutdowns {
            if s.Type == "restart" && s.Status == eshandler.ShutdownComplete {
                r.logger.Infof("Cleaning up completed shutdown for node %s", s.NodeID)
                if err := r.esHandler.DeleteNodeShutdown(ctx, s.NodeID); err != nil {
                    r.logger.Warnf("Failed to delete shutdown for node %s: %s", s.NodeID, err)
                }
            }
        }
        return nil
    case StrategyAllocationFilter:
        r.logger.Info("Re-enabling shard allocation (transient)")
        if err := r.esHandler.EnableShardAllocationTransient(); err != nil {
            return fmt.Errorf("failed to enable allocation: %w", err)
        }
        r.logger.Info("Re-enabling routing rebalance (transient)")
        if err := r.esHandler.EnableRoutingRebalanceTransient(); err != nil {
            return fmt.Errorf("failed to enable rebalance: %w", err)
        }
        return nil
    }
    return nil
}

// Request graceful shutdown for a pod (Node Shutdown API only)
func (r *RollingRestartOrchestrator) RequestNodeShutdown(ctx context.Context, podName string) error {
    if r.strategy != StrategyNodeShutdown {
        return nil  // No-op for allocation filter strategy
    }
    nodeID, ok := r.nodeNameToID[podName]
    if !ok {
        return fmt.Errorf("node ID not found for pod %s", podName)
    }
    r.logger.Infof("Requesting node shutdown for pod %s (node ID: %s)", podName, nodeID)
    return r.esHandler.PutNodeShutdown(ctx, nodeID, eshandler.ShutdownTypeRestart, "rolling-restart")
}

// Check if node shutdown is complete
func (r *RollingRestartOrchestrator) IsNodeShutdownComplete(ctx context.Context, podName string) (bool, error) {
    if r.strategy != StrategyNodeShutdown {
        return true, nil
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
        return true, nil
    }
    r.logger.Debugf("Node shutdown status for %s: %s", podName, info.Status)
    return info.Status == eshandler.ShutdownComplete, nil
}
```

---

### Phase 4: Safety Predicates (Adapted from ECK)

**File:** `internal/controller/elasticsearch/upgrade_predicates.go` (new)

**Simplified predicates for our operator (inspired by ECK, not copied):**

```go
// Predicate checks if it's safe to delete a pod during rolling upgrade
type UpgradePredicate struct {
    Name string
    Check func(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error)
}

// UpgradeState holds context for predicate evaluation
type UpgradeState struct {
    ES               *elasticsearchcrd.Elasticsearch
    ESHandler        eshandler.ElasticsearchHandler
    HealthyPods      map[string]corev1.Pod  // pod name -> pod (ready + in cluster)
    PodsToUpgrade    []corev1.Pod
    CurrentPods      []corev1.Pod
    ExpectedMasters  []string
    DeletedPods      []corev1.Pod
    Logger           *logrus.Entry
}

// Core predicates (adapted from ECK's approach)
var UpgradePredicates = []UpgradePredicate{
    {
        Name: "cluster_health_not_red",
        Check: func(ctx, pod, state) (bool, string, error) {
            health, err := state.ESHandler.ClusterHealth()
            if err != nil { return false, "", err }
            if health.Status == "red" {
                return false, "cluster health is RED", nil
            }
            return true, "", nil
        },
    },
    {
        Name: "require_started_replica_for_primary",
        Check: func(ctx, pod, state) (bool, string, error) {
            // Don't delete node with primary shard if no STARTED replica exists
            shardsByNode, err := state.ESHandler.GetShardsByNode(ctx)
            if err != nil { return false, "", err }
            shards := shardsByNode[pod.Name]
            for _, shard := range shards {
                if !shard.Primary { continue }
                _, started, err := state.ESHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, pod.Name)
                if err != nil { return false, "", err }
                if started == 0 {
                    total, _, _ := state.ESHandler.CountStartedReplicas(ctx, shard.Index, shard.Shard, "")
                    if total > 0 {
                        return false, fmt.Sprintf("no STARTED replica for primary shard %s/%s", shard.Index, shard.Shard), nil
                    }
                }
            }
            return true, "", nil
        },
    },
    {
        Name: "one_master_at_a_time",
        Check: func(ctx, pod, state) (bool, string, error) {
            // Only restart one master-eligible node at a time
            isMaster := IsMasterRole(state.ES, GetNodeGroupNameFromNodeName(pod.Name))
            if !isMaster { return true, "", nil }
            // Check if another master is already being deleted
            for _, deleted := range state.DeletedPods {
                deletedIsMaster := IsMasterRole(state.ES, GetNodeGroupNameFromNodeName(deleted.Name))
                if deletedIsMaster {
                    return false, "another master node is already being upgraded", nil
                }
            }
            return true, "", nil
        },
    },
    {
        Name: "skip_terminating_pods",
        Check: func(ctx, pod, state) (bool, string, error) {
            if pod.DeletionTimestamp != nil {
                return false, "pod is already terminating", nil
            }
            return true, "", nil
        },
    },
}

// ApplyPredicates runs all predicates on a candidate pod
func ApplyPredicates(ctx context.Context, pod corev1.Pod, state *UpgradeState) (bool, string, error) {
    for _, predicate := range UpgradePredicates {
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
```

**Annotation to disable predicates (like ECK):**
```go
const DisablePredicatesAnnotation = "elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates"
// Value: comma-separated predicate names, or "*" for all
```

---

### Phase 5: Pod Downtime/Suspend Feature (User's Key Request)

**Annotation:** `elasticsearch.k8s.webcenter.fr/suspend` (comma-separated pod names)

**How It Works (adapted from ECK's suspend feature):**

1. User annotates Elasticsearch CR:
   ```yaml
   metadata:
     annotations:
       elasticsearch.k8s.webcenter.fr/suspend: "my-es-data-0,my-es-data-1"
   ```

2. Operator creates/updates a ConfigMap `<es-name>-suspended-pods` with pod names

3. Each pod gets an init container that checks if its name is in the suspended list:
   ```yaml
   initContainers:
   - name: suspend-check
     image: busybox
     command: ["/bin/sh", "-c"]
     args:
       - |
         while grep -qx "$HOSTNAME" /suspended/suspended_pods.txt 2>/dev/null; do
           echo "Pod $HOSTNAME is suspended via elasticsearch.k8s.webcenter.fr/suspend annotation"
           echo "Remove the pod name from the annotation to resume"
           sleep 10
         done
     env:
     - name: HOSTNAME
       valueFrom:
         fieldRef:
           fieldPath: metadata.name
     volumeMounts:
     - name: suspended-pods
       mountPath: /suspended
   volumes:
   - name: suspended-pods
     configMap:
       name: <es-name>-suspended-pods
   ```

4. Operator's reconcile loop:
   - If pod is running but should be suspended → delete pod (it will restart and get suspended)
   - If pod is suspended but should not be → the init container will exit when ConfigMap updates
   - Log clearly: `"Pod my-es-data-0 is suspended (downtime mode)"`

**Implementation Files:**

**File:** `internal/controller/elasticsearch/pod_suspend.go` (new)
```go
package elasticsearch

const SuspendAnnotation = "elasticsearch.k8s.webcenter.fr/suspend"

// GetSuspendedPodNames returns the set of pod names to suspend
func GetSuspendedPodNames(es *elasticsearchcrd.Elasticsearch) map[string]bool {
    names := map[string]bool{}
    if val, ok := es.Annotations[SuspendAnnotation]; ok {
        for _, name := range strings.Split(val, ",") {
            name = strings.TrimSpace(name)
            if name != "" {
                names[name] = true
            }
        }
    }
    return names
}

// IsPodSuspended checks if a specific pod should be suspended
func IsPodSuspended(es *elasticsearchcrd.Elasticsearch, podName string) bool {
    return GetSuspendedPodNames(es)[podName]
}

// ReconcileSuspendedPods handles the pod suspension feature
// Inspired by ECK's suspend feature, adapted for our operator
func ReconcileSuspendedPods(ctx context.Context, c client.Client, es *elasticsearchcrd.Elasticsearch, logger *logrus.Entry) error {
    suspendedPods := GetSuspendedPodNames(es)
    logger.Infof("Pod suspension check: %d pods suspended (annotation: %s)", len(suspendedPods), SuspendAnnotation)

    // List all ES pods
    podList := &corev1.PodList{}
    labelSelectors, _ := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", es.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
    if err := c.List(ctx, podList, &client.ListOptions{Namespace: es.Namespace, LabelSelector: labelSelectors}); err != nil {
        return err
    }

    for i := range podList.Items {
        pod := &podList.Items[i]
        shouldSuspend := suspendedPods[pod.Name]
        isSuspended := isPodInSuspendedState(pod)

        switch {
        case shouldSuspend && !isSuspended && isPodRunning(pod):
            // Pod should be suspended but is running → delete it so it restarts and gets suspended
            logger.Infof("Suspending pod %s: deleting running pod so it restarts in suspended state", pod.Name)
            if err := c.Delete(ctx, pod, client.GracePeriodSeconds(0)); err != nil {
                return fmt.Errorf("failed to delete pod %s for suspension: %w", pod.Name, err)
            }

        case shouldSuspend && isSuspended:
            logger.Infof("Pod %s is suspended (downtime mode) - PVC remains mounted, use 'kubectl exec' to access data", pod.Name)

        case !shouldSuspend && isSuspended:
            // Pod is suspended but should not be → ConfigMap update will unblock init container
            logger.Infof("Pod %s is resuming from suspended state", pod.Name)
            // The init container will exit when ConfigMap is updated
        }
    }

    return nil
}

// isPodInSuspendedState checks if pod is stuck in init container (suspended)
func isPodInSuspendedState(pod *corev1.Pod) bool {
    for _, status := range pod.Status.InitContainerStatuses {
        if status.Name == "suspend-check" && status.State.Running != nil {
            return true
        }
    }
    return false
}

// isPodRunning checks if the main ES container is running
func isPodRunning(pod *corev1.Pod) bool {
    for _, status := range pod.Status.ContainerStatuses {
        if status.Name == "elasticsearch" && status.State.Running != nil {
            return true
        }
    }
    return false
}
```

**File:** `internal/controller/elasticsearch/configmap_builder.go` (add function)
```go
// buildSuspendedPodsConfigMap creates a ConfigMap with suspended pod names
func buildSuspendedPodsConfigMap(es *elasticsearchcrd.Elasticsearch) *corev1.ConfigMap {
    suspendedPods := GetSuspendedPodNames(es)
    names := make([]string, 0, len(suspendedPods))
    for name := range suspendedPods {
        names = append(names, name)
    }
    sort.Strings(names)

    return &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Namespace: es.Namespace,
            Name:      GetSuspendedPodsConfigMapName(es),
            Labels:    getLabels(es),
        },
        Data: map[string]string{
            "suspended_pods.txt": strings.Join(names, "\n"),
        },
    }
}

func GetSuspendedPodsConfigMapName(es *elasticsearchcrd.Elasticsearch) string {
    return fmt.Sprintf("%s-suspended-pods-es", es.Name)
}
```

**File:** `internal/controller/elasticsearch/statefulset_builder.go` (modify)
```go
// In buildStatefulsets(), add suspend init container and volume:

// Add suspend-check init container (before other init containers)
suspendContainer := corev1.Container{
    Name:  "suspend-check",
    Image: "busybox:latest",
    Command: []string{"/bin/sh", "-c"},
    Args: []string{
        `while grep -qx "$HOSTNAME" /suspended/suspended_pods.txt 2>/dev/null; do
  echo "Pod $HOSTNAME is suspended via ` + SuspendAnnotation + ` annotation"
  echo "Remove the pod name from the annotation to resume normal operation"
  sleep 10
done`,
    },
    Env: []corev1.EnvVar{
        {
            Name: "HOSTNAME",
            ValueFrom: &corev1.EnvVarSource{
                FieldRef: &corev1.ObjectFieldSelector{
                    FieldPath: "metadata.name",
                },
            },
        },
    },
    VolumeMounts: []corev1.VolumeMount{
        {
            Name:      "suspended-pods",
            MountPath: "/suspended",
            ReadOnly:  true,
        },
    },
}
ptb.WithInitContainers([]corev1.Container{suspendContainer}, k8sbuilder.Merge)

// Add suspended-pods volume
ptb.WithVolumes([]corev1.Volume{
    {
        Name: "suspended-pods",
        VolumeSource: corev1.VolumeSource{
            ConfigMap: &corev1.ConfigMapVolumeSource{
                LocalObjectReference: corev1.LocalObjectReference{
                    Name: GetSuspendedPodsConfigMapName(es),
                },
            },
        },
    },
}, k8sbuilder.Merge)
```

**Integration Point:**
Add to `elasticsearch_controller.go` in the reconcile loop (after statefulset reconciler):
```go
// After statefulset reconciliation, check for suspended pods
if err := ReconcileSuspendedPods(ctx, h.Client(), o, logger); err != nil {
    logger.Warnf("Error when reconciling suspended pods: %s", err)
}
```

---

### Phase 6: Update StatefulSet Reconciler

**File:** `internal/controller/elasticsearch/statefulset_reconciler.go`

**Changes in `Diff()` method:**

Replace the current simple rebalance disable/enable with the orchestrator:

```go
func (r *statefulsetReconciler) Diff(ctx context.Context, o *elasticsearchcrd.Elasticsearch, read multiphase.MultiPhaseRead[*appv1.StatefulSet], data map[string]any, logger *logrus.Entry) (diff multiphase.MultiPhaseDiff[*appv1.StatefulSet], res reconcile.Result, err error) {
    // ... existing code ...

    // When starting upgrade phase
    if data["phase"] == StatefulsetPhaseUpgradeStarted {
        // Create orchestrator
        orchestrator, err := NewRollingRestartOrchestrator(esHandler, r.Client(), logger)
        if err != nil {
            logger.Errorf("Failed to create rolling restart orchestrator: %s", err)
            return diff, res, errors.Wrap(err, "Error when create rolling restart orchestrator")
        }
        data["rollingRestartOrchestrator"] = orchestrator

        // Check predicates before starting
        predicateState := &UpgradeState{
            ES:              o,
            ESHandler:       esHandler,
            HealthyPods:     healthyPods,
            PodsToUpgrade:   podsToUpgrade,
            CurrentPods:     currentPods,
            ExpectedMasters: expectedMasters,
            Logger:          logger,
        }

        // For each pod that needs upgrade, check predicates
        for _, pod := range podsToUpgrade {
            ok, reason, err := ApplyPredicates(ctx, pod, predicateState)
            if err != nil {
                logger.Errorf("Error checking predicates for pod %s: %s", pod.Name, err)
                return diff, res, err
            }
            if !ok {
                logger.Warnf("Pod %s cannot be upgraded yet: %s", pod.Name, reason)
                // Skip this pod for now, will retry
                continue
            }
            // Pod passed predicates, can be upgraded
            logger.Infof("Pod %s passed all predicates, ready for upgrade", pod.Name)
        }

        // Prepare cluster for rolling restart
        if err := orchestrator.PrepareForRollingRestart(ctx); err != nil {
            logger.Warnf("Error preparing rolling restart (non-blocking): %s", err)
        }
    }

    // When finishing upgrade phase
    if data["phase"] == StatefulsetPhaseUpgradeFinished {
        if v, err := helper.Get(data, "rollingRestartOrchestrator"); err == nil {
            if orchestrator, ok := v.(*RollingRestartOrchestrator); ok {
                logger.Info("Completing rolling restart - re-enabling shard allocation")
                if err := orchestrator.CompleteRollingRestart(ctx); err != nil {
                    logger.Errorf("Error completing rolling restart: %s", err)
                    return diff, res, errors.Wrap(err, "Error when complete rolling restart")
                }
                logger.Info("Rolling restart completed successfully")
            }
        }
    }

    // ... rest of existing code ...
}
```

**Enhanced Logging (Info Level):**

Add detailed logging throughout the upgrade process:

```go
// At start of upgrade
logger.Infof("=== Rolling Upgrade Started ===")
logger.Infof("Cluster: %s/%s", o.Namespace, o.Name)
logger.Infof("StatefulSets to upgrade: %d", len(updates))
logger.Infof("Pods to upgrade: %d", len(podsToUpgrade))

// For each pod decision
logger.Infof("Pod %s: predicates passed, proceeding with upgrade", pod.Name)
logger.Infof("Pod %s: blocked by predicate '%s': %s", pod.Name, predicateName, reason)

// During upgrade
logger.Infof("Requesting node shutdown for pod %s (node ID: %s)", podName, nodeID)
logger.Infof("Node shutdown status for %s: %s", podName, status)
logger.Infof("Deleting pod %s for rolling upgrade", pod.Name)

// At completion
logger.Infof("=== Rolling Upgrade Completed ===")
logger.Infof("All pods upgraded successfully")
```

---

### Phase 7: Update StatefulSet Update Strategy

**File:** `internal/controller/elasticsearch/statefulset_builder.go`

**Current:** Uses default `RollingUpdate` strategy (K8s controls pod deletion)

**New:** Use `OnDelete` strategy (operator controls pod deletion)

```go
// In buildStatefulsets(), update the StatefulSet spec:
sts = &appv1.StatefulSet{
    // ... existing fields ...
    Spec: appv1.StatefulSetSpec{
        Replicas: ptr.To[int32](nodeGroup.Replicas),
        // Operator controls pod deletion for safer rolling upgrades
        UpdateStrategy: appv1.StatefulSetUpdateStrategy{
            Type: appv1.OnDeleteStatefulSetStrategyType,
        },
        PodManagementPolicy: appv1.ParallelPodManagement,
        ServiceName:         GetNodeGroupServiceNameHeadless(es, nodeGroup.Name),
        // ... rest of spec ...
    },
}
```

**Why:** With `OnDelete`, the StatefulSet controller won't automatically delete pods. Our operator's rolling restart orchestrator will delete pods one at a time with proper safety checks.

**Migration Note:** Existing StatefulSets will be updated from `RollingUpdate` to `OnDelete`. This is safe because:
- The update strategy change doesn't affect existing pods
- Only affects future pod updates
- Our orchestrator takes over pod deletion logic

---

### Phase 8: Tests

**Files to create/modify:**

1. **`internal/controller/elasticsearch/rolling_restart_test.go`** (new)
   - Test orchestrator strategy selection (ES version detection)
   - Test PrepareForRollingRestart for both strategies
   - Test CompleteRollingRestart
   - Test predicate evaluation

2. **`internal/controller/elasticsearch/pod_suspend_test.go`** (new)
   - Test GetSuspendedPodNames parsing
   - Test ReconcileSuspendedPods logic
   - Test ConfigMap generation

3. **`internal/controller/elasticsearch/upgrade_predicates_test.go`** (new)
   - Test each predicate individually
   - Test predicate combinations
   - Test annotation-based predicate disabling

4. **`internal/controller/elasticsearch/elasticsearch_controller_test.go`** (modify)
   - Test headless service DNS address construction
   - Test multiple address failover

5. **`internal/controller/elasticsearch/statefulset_reconciler_test.go`** (modify)
   - Test integration with orchestrator
   - Test phase transitions

6. **`/projects/es-handler/*_test.go`** (new/modify)
   - Test all new es-handler methods

---

### Phase 9: Documentation

**Files to create/update:**

1. **`documentations/rolling-upgrade.md`** (new)
   - How rolling upgrades work
   - Node Shutdown API vs allocation filtering
   - Safety predicates explained
   - Troubleshooting guide

2. **`documentations/pod-downtime.md`** (new)
   - How to use the suspend annotation
   - Use cases (PVC access, debugging, crashback recovery)
   - Examples

3. **`README.md`** (update)
   - Mention new features

---

## Implementation Order

| Phase | Description | Priority | Dependencies |
|-------|-------------|----------|--------------|
| 1 | ES-Handler enhancements | **CRITICAL** | None |
| 2 | Smart ES client connection (headless DNS) | **CRITICAL** | Phase 1 |
| 3 | Rolling restart orchestrator | High | Phase 1 |
| 4 | Safety predicates | High | Phase 1, 3 |
| 5 | Pod downtime/suspend feature | **HIGH** (user request) | None |
| 6 | Update statefulset reconciler | High | Phase 2, 3, 4 |
| 7 | Update StatefulSet strategy to OnDelete | Medium | Phase 6 |
| 8 | Tests | High | All phases |
| 9 | Documentation | Medium | All phases |

**Recommended order:**
1. Phase 1 (es-handler) - blocking dependency
2. Phase 2 (smart connection) - critical for everything else
3. Phase 5 (pod suspend) - user's key request, independent
4. Phase 3+4 (orchestrator + predicates) - core upgrade logic
5. Phase 6+7 (integrate into reconciler)
6. Phase 8 (tests)
7. Phase 9 (docs)

---

## Data Safety Guarantees

### PVC Preservation

| Aspect | ECK | Our Operator | Compatible? |
|--------|-----|--------------|-------------|
| VolumeClaimTemplate name | `elasticsearch-data` | `elasticsearch-data` | ✅ YES |
| PVC naming pattern | `<vct>-<sts>-<ordinal>` | `<vct>-<sts>-<ordinal>` | ✅ YES |
| StatefulSet naming | `<es-name>-<nodeset>-es` | `<es-name>-<nodegroup>-es` | ⚠️ DIFFERENT |

**Important:** While the VCT name is the same, the StatefulSet naming might differ:
- ECK: `es-name-nodeset-name-es-<ordinal>` (e.g., `my-es-es-master-0`)
- Ours: `es-name-nodegroup-name-es-<ordinal>` (e.g., `my-es-master-es-0`)

**This doesn't matter for PVC preservation** because:
- PVCs are owned by StatefulSets
- If we keep our existing StatefulSets (just update their spec), PVCs are preserved
- We're NOT migrating from ECK, we're enhancing our existing operator
- Existing PVCs from our operator will continue to work

### Migration Safety

Since we're enhancing the existing operator (not migrating from ECK):

1. **Existing StatefulSets** remain unchanged (same names)
2. **Existing PVCs** remain unchanged (same names, same data)
3. **Update strategy change** (RollingUpdate → OnDelete) is safe:
   - Doesn't affect existing pods
   - Only affects future pod recreations
4. **New init container** (suspend-check) is added:
   - Runs before ES starts
   - Exits immediately if pod not in suspend list
   - No impact on existing running pods until they restart

---

## Logging Requirements (Info Level)

The user specifically requested: **"clear log on info level or more to understand why pod is not reconciled"**

### Logging Matrix

| Scenario | Log Level | Message Example |
|----------|-----------|-----------------|
| Pod suspended (downtime) | INFO | `Pod my-es-data-0 is suspended (downtime mode via elasticsearch.k8s.webcenter.fr/suspend)` |
| Pod resuming from suspend | INFO | `Pod my-es-data-0 resuming from suspended state` |
| Rolling upgrade started | INFO | `=== Rolling Upgrade Started: 3 pods to upgrade across 2 StatefulSets ===` |
| Predicate blocked pod | INFO | `Pod my-es-data-1 blocked by predicate 'require_started_replica_for_primary': no STARTED replica for shard logs-000001/0` |
| Predicate passed | INFO | `Pod my-es-data-0 passed all safety predicates` |
| Node shutdown requested | INFO | `Requesting graceful shutdown for pod my-es-data-0 (node ID: abc123)` |
| Node shutdown status | DEBUG | `Node shutdown status for my-es-data-0: IN_PROGRESS` |
| Node shutdown complete | INFO | `Node shutdown complete for pod my-es-data-0, proceeding with pod deletion` |
| Pod deleted for upgrade | INFO | `Deleting pod my-es-data-0 for rolling upgrade (graceful shutdown complete)` |
| Transient settings set | INFO | `Set transient cluster.routing.allocation.enable=primaries (replica allocation disabled)` |
| Transient settings cleared | INFO | `Cleared transient allocation settings, shard allocation re-enabled` |
| Rolling upgrade completed | INFO | `=== Rolling Upgrade Completed: all 3 pods upgraded successfully ===` |
| ES client addresses | INFO | `Elasticsearch client configured with 7 addresses (6 pod DNS + 1 service)` |
| ES client failover | WARN | `Failed to connect to my-es-data-0.my-es-data-headless-es.ns.svc:9200, trying next address` |

### Structured Logging Fields

Use logrus fields for better log aggregation:

```go
logger.WithFields(logrus.Fields{
    "es_name":      o.Name,
    "namespace":    o.Namespace,
    "pod_name":     pod.Name,
    "statefulset":  sts.Name,
    "node_group":   nodeGroup.Name,
    "es_version":   version,
    "strategy":     strategy,
    "phase":        phase,
}).Info("Rolling upgrade progress")
```

---

## User's Specific Requirements Checklist

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| Clone ECK and analyze | ✅ DONE | ECK at `/tmp/eck`, analysis complete |
| Understand ECK's lifecycle (not StatefulSet but pods) | ✅ CLARIFIED | ECK **DOES** use StatefulSets, not direct pods |
| Backport if gap is enough | ✅ PLANNED | Phases 2-7 above |
| Keep existing data (PVC names same) | ✅ GUARANTEED | Both use `elasticsearch-data` VCT name |
| Clear log on info level | ✅ PLANNED | Logging matrix above (Phase 6) |
| Annotation for pod downtime | ✅ PLANNED | `elasticsearch.k8s.webcenter.fr/suspend` (Phase 5) |
| Stop pod without recreating it | ✅ PLANNED | Suspend feature keeps pod in Init state, PVC mounted |
| Access inside PVC when crashback | ✅ SUPPORTED | Suspended pod allows `kubectl exec` to access data |
| Don't copy ECK code directly | ✅ RESPECTED | Adapt concepts, write original code, add attribution |
| Look at optimize-rolling-restart-shard-allocation.md | ✅ DONE | Referenced and incorporated into this plan |

---

## Files to Create/Modify Summary

### es-handler (separate repo)
| File | Action | Description |
|------|--------|-------------|
| `cluster.go` | Modify | Add transient settings, GetVersion, Flush |
| `shutdown.go` | Create | Node Shutdown API methods |
| `nodes.go` | Create | Node info methods |
| `shard.go` | Create | Shard listing methods |
| `elasticsearch.go` | Modify | Update interface |
| `mocks/elasticsearch_handler.go` | Regenerate | Update mocks |
| `*_test.go` | Create/Modify | Tests for new methods |

### elasticsearch-operator
| File | Action | Description |
|------|--------|-------------|
| `internal/controller/elasticsearch/elasticsearch_controller.go` | Modify | Smart ES client connection (headless DNS) |
| `internal/controller/elasticsearch/rolling_restart.go` | Create | Rolling restart orchestrator |
| `internal/controller/elasticsearch/upgrade_predicates.go` | Create | Safety predicates |
| `internal/controller/elasticsearch/pod_suspend.go` | Create | Pod downtime/suspend feature |
| `internal/controller/elasticsearch/statefulset_reconciler.go` | Modify | Integrate orchestrator + predicates |
| `internal/controller/elasticsearch/statefulset_builder.go` | Modify | Add suspend init container, OnDelete strategy |
| `internal/controller/elasticsearch/configmap_builder.go` | Modify | Add suspended pods ConfigMap builder |
| `internal/controller/elasticsearch/helper.go` | Modify | Add suspend-related helpers |
| `api/elasticsearch/v1/elasticsearch_types.go` | Modify | Add SuspendAnnotation constant |
| `internal/controller/elasticsearch/*_test.go` | Create/Modify | Tests |
| `documentations/rolling-upgrade.md` | Create | Rolling upgrade docs |
| `documentations/pod-downtime.md` | Create | Pod downtime docs |

---

## Validation Plan

### Unit Tests
```bash
# es-handler tests
cd /projects/es-handler && go test ./...

# Operator tests
cd /projects/elasticsearch-operator && go test ./internal/controller/elasticsearch/...
```

### Integration Test Scenario

1. **Test Pod Suspension:**
   ```bash
   # Create cluster
   kubectl apply -f test-cluster.yaml

   # Suspend a pod
   kubectl annotate es my-es elasticsearch.k8s.webcenter.fr/suspend="my-es-data-0"

   # Verify pod is suspended
   kubectl get pods -l cluster=my-es
   # my-es-data-0 should be in Init state

   # Access PVC data
   kubectl exec -it my-es-data-0 -c suspend-check -- /bin/sh
   # Can access /usr/share/elasticsearch/data (mounted PVC)

   # Resume pod
   kubectl annotate es my-es elasticsearch.k8s.webcenter.fr/suspend-
   # Pod should start normally
   ```

2. **Test Rolling Upgrade:**
   ```bash
   # Create 3-node cluster
   kubectl apply -f test-cluster-3nodes.yaml

   # Create index with replica
   curl -X PUT "https://my-es:9200/logs" -d '{"settings":{"number_of_replicas":1}}'

   # Trigger rolling upgrade (change image)
   kubectl patch es my-es --type=merge -p '{"spec":{"version":"8.11.0"}}'

   # Watch logs
   kubectl logs -f deploy/elasticsearch-operator -n elasticsearch-operator

   # Verify:
   # - INFO logs show rolling upgrade progress
   # - Only one pod upgraded at a time
   # - Node shutdown API used (ES >= 7.15.2)
   # - No data loss
   # - Cluster health returns to green
   ```

### Manual Verification Commands

```bash
# Check operator connection method
kubectl logs deploy/elasticsearch-operator | grep "Elasticsearch client configured"

# Check suspended pods
kubectl get es my-es -o jsonpath='{.metadata.annotations.elasticsearch\.k8s\.webcenter\.fr/suspend}'

# Check pod suspension status
kubectl describe pod my-es-data-0 | grep -A5 "suspend-check"

# Check transient settings during upgrade
kubectl exec -it my-es-master-0 -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_cluster/settings?include_defaults=true | jq '.transient'

# Check node shutdown status
kubectl exec -it my-es-master-0 -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_nodes/shutdown

# Check shard allocation
kubectl exec -it my-es-master-0 -- curl -k -u elastic:$ELASTIC_PASSWORD \
  https://localhost:9200/_cat/shards?v
```

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Headless DNS not resolving | Operator can't connect | Verify headless service has `PublishNotReadyAddresses: true` (already set) |
| Too many addresses for large clusters | ES client slow | Limit to first N pods, or use global headless service |
| Transient settings lost on cluster restart | Rebalancing stays disabled | Acceptable - settings are meant to be temporary |
| Node Shutdown API not available | Fall back needed | Auto-detect ES version, use allocation filter strategy |
| Operator crashes during upgrade | Cluster state inconsistent | Transient settings auto-reset; Node Shutdown API is self-healing |
| Suspend annotation misconfigured | Pod stuck forever | Clear error message in logs; easy to fix by removing annotation |
| OnDelete strategy blocks updates | Pods not updated | Orchestrator must handle pod deletion; add timeout/fallback |
| Predicate too strict | Upgrade blocked | Annotation to disable specific predicates |

---

## Open Questions

1. **Should we support `maxUnavailable` budget like ECK?**
   - ECK has `spec.updateStrategy.changeBudget.maxUnavailable`
   - Our operator currently upgrades one StatefulSet at a time
   - **Recommendation:** Add in future enhancement, not in initial backport

2. **Should we support pre-stop hooks for graceful ES shutdown?**
   - ECK has lifecycle hooks
   - Node Shutdown API makes this less critical
   - **Recommendation:** Add if Node Shutdown API is not available (ES < 7.15.2)

3. **Should the suspend feature support a timeout?**
   - Auto-resume after X hours?
   - **Recommendation:** No, keep it simple - manual control via annotation

4. **Should we add metrics for rolling upgrades?**
   - Prometheus metrics for upgrade duration, pods upgraded, etc.
   - **Recommendation:** Future enhancement

---

## References

- **ECK Repository:** https://github.com/elastic/cloud-on-k8s (cloned at `/tmp/eck`)
- **ECK License:** Elastic License 2.0
- **Our License:** Apache License 2.0
- **Existing Plan:** `.opencode/plans/optimize-rolling-restart-shard-allocation.md`
- **ES-Handler Plan:** `.opencode/plans/es-handler-rolling-restart-enhancements.md`
- **Elasticsearch Node Shutdown API:** https://www.elastic.co/guide/en/elasticsearch/reference/current/nodes-shutdown-apis.html
- **Elasticsearch Rolling Upgrades:** https://www.elastic.co/guide/en/elasticsearch/reference/current/rolling-upgrades.html

---

## Attribution

This plan is inspired by the architecture of ECK (Elastic Cloud on Kubernetes) by Elastic B.V.
The concepts have been adapted for our operator's architecture and Apache 2.0 license.
No ECK code has been copied directly.

ECK is licensed under the Elastic License 2.0.
Our operator is licensed under the Apache License 2.0.
