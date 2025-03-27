# Node settings groups

You should to provide one or more node groups. A node groups is Elasticsearch nodes that share the same settings. For basic topology, you will create 3 nodes groups:
- client: then endpoint nodes
- master: the master nodes
- data: the data nodes

> The node group settings will be merged with the global settings.

You can use the following setting for each node group:
- **name** (string / required): the node group name.
- **replicas** (number / required): The number of instances
- **roles** (slice of string / required): The list of node roles
- **persistence** (object): The persistent volume to use to store Elasticsearch data. Default is `emptyDir` (not peristent)
  - **volumeClaim** (object): Use it if you should to use PVC. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
  - **volume** (object): Use it if you should to use existing volume or hostPath. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it. Default is `soft` with the key `kubernetes.io/hostname`
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **resources** (object): The default resources for Elasticsearch pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **jvm** (string): Set additionnal JVM options. Default is `empty`.
- **config** (map of any): The config of Elasticsearch on YAML format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on config folder. Each value is the file contend. It permit to set elasticsearch.yml settings. Default is `empty`.
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod per node groups. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **podTemplate** (object): The pod template to merge with the Elasticsearch pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **waitClusterStatus** (string): Wait the cluster status provided for readiness check. Default to `green`.
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)


**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  labels:
    socle: cluster-dev
  name: elasticsearch
  namespace: cluster-dev
spec:
  nodeGroups:
    - name: master
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Gi
          storageClassName: openebs-hostpath
      replicas: 3
      resources:
        limits:
          cpu: '2'
          memory: 4Gi
        requests:
          cpu: '2'
          memory: 4Gi
      roles:
        - master
    - name: data
      nodeSelector:
        project: cluster-prd-data
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1300Gi
          storageClassName: openebs-hostpath
      replicas: 5
      podDisruptionBudget:
        maxUnavailable: 2
      resources:
        limits:
          cpu: '11'
          memory: 62Gi
        requests:
          cpu: '11'
          memory: 62Gi
      roles:
        - data_hot
        - data_content
      tolerations:
        - effect: NoSchedule
          key: project
          operator: Equal
          value: cluster-prd-hot
      config:
        node.attr.data: "warm"
      antiAffinity:
        topologyKey: topology.kubernetes.io/zone
        type: hard
    - name: client
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Gi
          storageClassName: openebs-hostpath
      replicas: 2
      resources:
        limits:
          cpu: '2'
          memory: 4Gi
        requests:
          cpu: '2'
          memory: 4Gi
      roles:
        - ingest
      waitClusterStatus: yellow
      podTemplate:
        spec:
          containers:
            - name: elasticsearch
                command:
                  - custom command
      jvm: '-Xmx1G - Xms1G'
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