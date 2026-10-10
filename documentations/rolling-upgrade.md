# Rolling upgrade

The operator upgrades Elasticsearch clusters with a rolling upgrade: pods are
restarted **one StatefulSet at a time**, and within a StatefulSet the
StatefulSet controller rolls pods one by one (highest ordinal first). The
whole process is gated by the `StatefulsetUpgrade` status condition so that
only one StatefulSet is upgraded at any time.

## How the rolling upgrade works

1. The operator detects that one or more StatefulSets need an update (new
   image, config change, TLS renewal, keystore change, etc.).
2. Before applying the first update, it evaluates the safety predicates (see
   below). If a predicate blocks the upgrade, the operator retries every 30
   seconds until the cluster is safe.
3. It prepares the cluster for the rolling restart using a version-aware
   strategy (see below).
4. It applies the update to **one** StatefulSet and sets the
   `StatefulsetUpgrade` condition to `True`. While this condition is `True`
   and the `StatefulsetReady` condition is `False`, only the StatefulSet
   currently being upgraded (or StatefulSets with 0 replica) is allowed to
   change — the other StatefulSets wait.
5. Once all pods of the upgraded StatefulSet are ready and no other
   StatefulSet needs an update, the operator completes the rolling restart
   (re-enables shard allocation/rebalancing or cleans up node shutdown
   requests) and resets the `StatefulsetUpgrade` condition.

## Version-aware strategy

The operator detects the Elasticsearch version and selects the strategy used
to prepare the cluster before restarting pods:

- **Elasticsearch >= 7.15.2** (`node-shutdown` strategy): the operator uses
  the [Node Shutdown API](https://www.elastic.co/guide/en/elasticsearch/reference/current/node-shutdown.html).
  When a `restart` shutdown request is registered for a node, Elasticsearch
  migrates shards away from it gracefully. **No cluster settings are
  changed.**

  > **Note:** the operator-driven pod deletion loop (register a `restart`
  > shutdown request before deleting a pod, wait for its completion, clear it
  > once the pod is back) is not enabled yet: StatefulSets still use the
  > `RollingUpdate` strategy, so the StatefulSet controller restarts the pods
  > one by one. The operator cleans up any completed `restart` shutdown
  > requests when the rolling upgrade finishes.
- **Elasticsearch < 7.15.2** (`allocation-filter` strategy): the operator sets
  transient cluster settings before the rolling restart and re-enables them
  afterwards:
  - `cluster.routing.allocation.enable=primaries` (replica shard allocation is
    disabled while pods are restarted)
  - `cluster.routing.rebalance.enable=none` (shard rebalancing is disabled)
  - a flush of all indices to speed up recovery

  These settings are **transient**: they live in the cluster state only and
  are automatically reset when the cluster restarts, so they never persist
  across a full cluster restart.

## Operator connectivity during an upgrade

The operator connects to the Elasticsearch API using one DNS address per pod,
built from the headless services:

```
https://<pod-name>.<es-name>-<node-group>-headless-es.<namespace>.svc:9200
```

Headless services publish pod DNS records even when pods are **not ready**, so
the operator can always reach a node — including during a rolling upgrade
when pods are being restarted and are not ready yet. The global ClusterIP
service (`<es-name>-es.<namespace>.svc:9200`) is appended as a fallback; it
load-balances across ready pods only.

## Safety predicates

Before upgrading a StatefulSet, the operator evaluates safety predicates. If
any predicate blocks the upgrade, the operator logs the reason and retries
every 30 seconds.

| Predicate | Blocks when |
|---|---|
| `cluster_health_not_red` | The cluster health is `red`. |
| `require_started_replica_for_primary` | A pod to upgrade hosts a primary shard that has replicas configured but no `STARTED` replica on another node. |
| `one_master_at_a_time` | Another master-eligible node is already being upgraded (terminating). |
| `skip_terminating_pods` | The pod is already terminating. |

### Disabling predicates

You can disable predicates individually (or all at once) with the
`elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates` annotation on the
`Elasticsearch` resource. The value is a comma-separated list of predicate
names, or `*` to disable all of them:

```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch
  annotations:
    elasticsearch.k8s.webcenter.fr/disable-upgrade-predicates: "cluster_health_not_red,one_master_at_a_time"
```

> Disabling predicates can cause data loss or cluster unavailability during
> an upgrade. Use with caution, and prefer fixing the underlying condition
> (e.g. wait for the cluster health to recover).

## Logs

During a rolling upgrade, the operator logs the following at INFO level:

- `=== Rolling Upgrade Started: StatefulSet <name> ===` — the upgrade of a
  StatefulSet starts.
- `Preparing for rolling restart using strategy: <strategy> (ES version: <version>)`
  — the selected strategy.
- `Set transient cluster.routing.allocation.enable=primaries ...` /
  `Set transient cluster.routing.rebalance.enable=none ...` /
  `Flushing indices for faster recovery` — legacy strategy preparation
  (ES < 7.15.2 only).
- `Using Node Shutdown API strategy - no cluster settings to change` — Node
  Shutdown API strategy (ES >= 7.15.2 only).
- `StatefulSet <name> passed all safety predicates (<n> pod(s) checked)` — all
  predicates passed.
- `Pod <name> blocked by predicate '<name>': <reason>` — a predicate blocked
  a pod.
- `Rolling upgrade of StatefulSet <name> blocked by safety predicates: <reason>`
  — a cluster-level predicate blocked the StatefulSet.
- `Requesting graceful shutdown for pod <name> (node ID: <id>)` — a node
  shutdown request was sent (ES >= 7.15.2 only).
- `Clearing node shutdown for pod <name> (node ID: <id>)` — a completed node
  shutdown request was cleaned up.
- `Re-enabling shard allocation (transient)` /
  `Re-enabling routing rebalance (transient)` — legacy strategy cleanup.
- `=== Rolling Upgrade Completed: completing rolling restart ===` — the
  rolling upgrade is finished.
- `Elasticsearch client configured with <n> addresses (<n-1> pod DNS via headless services + 1 global service)`
  — the operator's Elasticsearch client addresses.

## Troubleshooting

### The upgrade is delayed by predicates

Check the operator logs for `blocked by predicate` or `blocked by safety
predicates` messages — they include the predicate name and the reason.
Common causes:

- Cluster health is `red`: check unassigned shards with
  `GET _cluster/health` and `GET _cat/shards?v&h=index,shard,prirep,state,node`.
- No `STARTED` replica for a primary: wait for replicas to be allocated and
  started, or check disk watermarks / shard allocation with
  `GET _cluster/allocation/explain`.

The upgrade automatically retries every 30 seconds; no action is needed once
the blocking condition is resolved.

### A node shutdown is stuck

On ES >= 7.15.2, inspect the node shutdown requests:

```bash
curl -u elastic:<password> https://<es-name>-es.<namespace>.svc:9200/_nodes/shutdown
```

A shutdown stuck in `IN_PROGRESS` usually means shard migration cannot
complete (e.g. not enough disk space on the remaining nodes, or allocation
filters). You can cancel a shutdown request with:

```bash
curl -X DELETE -u elastic:<password> https://<es-name>-es.<namespace>.svc:9200/_nodes/shutdown/<node-id>
```

### Inspect the transient allocation settings

On ES < 7.15.2, check the transient settings during an upgrade:

```bash
curl -u elastic:<password> "https://<es-name>-es.<namespace>.svc:9200/_cluster/settings?include_defaults=true&filter_path=transient"
```

During an upgrade you should see:

```json
{
  "transient": {
    "cluster.routing.allocation.enable": "primaries",
    "cluster.routing.rebalance.enable": "none"
  }
}
```

They are automatically reset to `all` when the rolling upgrade completes,
and they disappear entirely after a full cluster restart (transient settings
are not persisted).
