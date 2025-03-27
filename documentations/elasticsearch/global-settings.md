# Global settings shared by node groups on Elasticsearch

When you deploy Elasticsearch cluster, you will create different kind of Elasticsearch nodes called here `node groups` like client, master and data. 
Some time you need to share same settings like PVC to store snapshot or elasticsearch.yaml settings.

You can use the following global setting:
- **additionalVolumes** (slice of object): Attach additional volume on Elasticsearch pods. Default to `empty`
  - **name** (string / required): The volume name
  - **volumeMount** (string / required): The path to mount the volume inside the pod
  - **volumeSource** (object / required): The pod volume source. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it.
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod per node groups. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **initContainerResources** (object): The default resources for all init containers. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podTemplate** (object): The pod template to merge with the Elasticsearch pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **jvm** (string): Set additionnal JVM options. Default is `empty`.
- **config** (map of any): The config of Elasticsearch on YAML format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on config folder. Each value is the file contend. It permit to set elasticsearch.yml settings. Default is `empty`.
- **keystoreSecretRef** (object): The secrets to inject on keystore on runtime. Each keys / values is injected on Java Keystore. Default to `empty`.
  - **name** (string / required): The secret name.
- **caSecretRef** (object): The custom CA to ijnect on Java cacerts on runtime. The key is the alias and the contend is the certificate contend. Default to `empty`.
  - **name** (string / required): The secret name.
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)


**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
    socle: cluster-dev
  name: elasticsearch
  namespace: cluster-dev
spec:
  globalNodeGroup:
    additionalVolumes:
      - mountPath: /mnt/snapshot
        name: snapshot
        persistentVolumeClaim:
          claimName: pvc-elasticsearch-snapshot
    antiAffinity:
      topologyKey: topology.kubernetes.io/zone
      type: hard
    podDisruptionBudget:
      maxUnavailable: 2
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    podTemplate:
      spec:
        containers:
          - name: elasticsearch
            command:
              - custom command
    jvm: '-Xmx1G - Xms1G'
    config:
      action.destructive_requires_name: true
      gateway.recover_after_time: 5m
      http.cors.allow-credentials: true
      http.cors.allow-headers: X-Requested-With,X-Auth-Token,Content-Type,
      Content-Length, Authorization
      http.cors.allow-origin: /.*/
      http.cors.enabled: true
      http.max_content_length: 500mb
      # Rack awarness
      cluster.routing.allocation.awareness.attributes: node_name
      # Repository
      path.repo:
        - /mnt/snapshot
      xpack.security.audit.enabled: true
      xpack.security.audit.logfile.events.exclude:
        - access_granted
      xpack.security.authc:
        anonymous:
          authz_exception: false
          roles: monitoring
          username: anonymous_user
        realms:
          active_directory.active_directory:
            order: 2
            url:
              - "ldap://dc1.domain.local:389"
              - "ldap://dc2.domain.local:389"
            load_balance.type: "failover"
            follow_referrals: true
            bind_dn: "${ELASTICSEARCH_LDAP_USER}"
            timeout.ldap_search: 60s
            domain_name: DOMAIN
            user_search:
              base_dn: "DC=DOMAIN,DC=LOCAL"
              scope: sub_tree
            group_search:
              base_dn: "OU=Users,DC=DOMAIN,DC=LOCAL"
              scope: sub_tree
            unmapped_groups_as_roles: false
      # Custom config
      gateway.expected_data_nodes: 3
      gateway.recover_after_data_nodes: 2
      cluster.routing.allocation.disk.watermark.low: 50gb
      cluster.routing.allocation.disk.watermark.high: 20gb
      cluster.routing.allocation.disk.watermark.flood_stage: 10gb
    keystoreSecretRef:
      name: elasticsearch-keystore
    caSecretRef:
      name: custom-ca
    labels:
      label1: my label
    annotations:
      annotation1: my annotation
    envFrom:
      - secretRef:
          name: elasticsearch-env
    env:
      - name: LDAP_USERNAME
        value: ldap_user
```

**pvc-elasticsearch-snapshot.yaml**:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-elasticsearch-snapshot
  namespace: cluster-dev
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
  storageClassName: nfs-client
```

**elasticsearch-keystore-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-keystore
  namespace: cluster-dev
type: Opaque
data:
  xpack.security.authc.realms.active_directory.active_directory.secure_bind_password: ++++++++
```

**custom-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca
  namespace: cluster-dev
type: Opaque
data:
  custom-ca.crt: ++++++++
```

**elasticsearch-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-env
  namespace: cluster-dev
type: Opaque
data:
  ELASTICSEARCH_LDAP_USER: ++++++++
```