# Pod suspend (downtime mode)

The operator can suspend individual Elasticsearch pods — a "downtime mode"
inspired by ECK (Elastic Cloud on Kubernetes). A suspended pod is kept in
`Init` state with its PVC still mounted, so you can inspect its data with
`kubectl exec` while the node is out of the cluster.

## How to suspend pods

Set the `elasticsearch.k8s.webcenter.fr/suspend` annotation on the
`Elasticsearch` resource with a comma-separated list of pod names:

```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch
  annotations:
    elasticsearch.k8s.webcenter.fr/suspend: "elasticsearch-master-es-0"
```

Pod names follow the pattern `<es-name>-<node-group>-es-<ordinal>`, e.g.
`elasticsearch-master-es-0`, `elasticsearch-data-es-1`.

To suspend several pods, separate the names with commas:

```yaml
metadata:
  annotations:
    elasticsearch.k8s.webcenter.fr/suspend: "elasticsearch-data-es-0, elasticsearch-data-es-1"
```

## How it works

1. The operator creates a ConfigMap named `<es-name>-suspended-pods-es`
   containing the suspended pod names (one per line, in the
   `suspended_pods.txt` key). This ConfigMap is **not** part of the
   StatefulSet checksum, so changing the suspend list does not trigger a
   rolling restart.
2. Every Elasticsearch pod runs a `suspend-check` init container (first in
   the init container list). It loops while the pod's own name (`$HOSTNAME`)
   is listed in `/suspended/suspended_pods.txt`:

   ```sh
   while grep -qx "$HOSTNAME" /suspended/suspended_pods.txt 2>/dev/null; do
     echo "Pod $HOSTNAME is suspended via elasticsearch.k8s.webcenter.fr/suspend annotation"
     echo "Remove the pod name from the annotation to resume normal operation"
     sleep 10
   done
   ```

3. When you add a pod name to the annotation, the operator gracefully deletes
   the running pod. It restarts and is blocked by the `suspend-check` init
   container: the pod stays in `Init` state and the Elasticsearch container
   never starts.
4. The init container mounts the Elasticsearch data volume
   (`elasticsearch-data`) **read-only** at `/usr/share/elasticsearch/data`,
   so the PVC content stays intact and can be inspected.
5. To resume a pod, remove its name from the annotation (or remove the
   annotation entirely). The operator updates the
   `<es-name>-suspended-pods-es` ConfigMap; the init container exits within
   ~10 seconds and the pod starts normally.

## Usage

Suspend one pod:

```bash
kubectl annotate elasticsearch elasticsearch \
  elasticsearch.k8s.webcenter.fr/suspend="elasticsearch-master-es-0"
```

Check the pod is suspended (it stays in `Init`):

```bash
kubectl get pods -l cluster=elasticsearch
# elasticsearch-master-es-0   0/1     Init:0/1   0   5m
```

Inspect the data on the PVC while the pod is suspended (exec into the
`suspend-check` init container, which mounts the data volume read-only):

```bash
kubectl exec -it elasticsearch-master-es-0 -c suspend-check -- /bin/sh
# ls /usr/share/elasticsearch/data
```

Resume the pod by removing the annotation:

```bash
kubectl annotate elasticsearch elasticsearch \
  elasticsearch.k8s.webcenter.fr/suspend-
```

Or edit the `Elasticsearch` resource and remove the pod name from the list to
resume a single pod while keeping the others suspended.

## Use cases

- **Access PVC data**: exec into a suspended pod to inspect, copy or debug
  the data directory without the Elasticsearch process running.
- **Debug a crashlooping pod**: suspend the pod to keep it (and its PVC)
  around, then exec into it to inspect logs, config or data.
- **Crashback recovery**: after a node crash, suspend the pod to prevent it
  from rejoining the cluster with a corrupted state, fix the data, then
  resume it.

## Caveats

- A suspended pod **does not serve any traffic**: it is not part of the
  cluster, does not hold shards, and is not ready. The headless service DNS
  record still resolves (headless services publish not-ready addresses), but
  connections to the pod will fail.
- Before suspending a pod, check the cluster health and replica availability:
  suspending a node that holds the only copy of some shards makes those
  shards unavailable until the pod is resumed.

  ```bash
  kubectl exec -it <any-es-pod> -- curl -s localhost:9200/_cluster/health?pretty
  ```

- The suspend list is stored in the `<es-name>-suspended-pods-es` ConfigMap.
  Do not edit it directly — always use the annotation on the `Elasticsearch`
  resource.
