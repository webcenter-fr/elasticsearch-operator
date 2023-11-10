# Deployment settings for Logstash


You can use the following setting to drive how to deploy Logstash:
- **replicas** (number / required): The number of instances
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it. Default is `soft` with the key `kubernetes.io/hostname`
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **resources** (object): The default resources for Logstash pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **podTemplate** (object): The pod template to merge with the Logstash pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- **initContainerResources** (object): The default resources for all init containers. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **jvm** (string): Set additionnal JVM options. Default is `empty`.
- **additionalVolumes** (slice of object): Attach additional volume on Logstash pods. Default to `empty`
  - **name** (string / required): The volume name
  - **volumeMount** (string / required): The path to mount the volume inside the pod
  - **volumeSource** (object / required): The pod volume source. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **persistence** (object): The persistent volume to use to store Elasticsearch data. Default is `emptyDir` (not peristent)
  - **volumeClaim** (object): Use it if you should to use PVC. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
  - **volume** (object): Use it if you should to use existing volume or hostPath. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **ports** (slice of object): It permit to set container ports on Pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)


**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  deployment:
    annotations:
      annotation1: my annotation
    labels:
      label1: my label
    env:
      - name: ENV1
        value: 'value1'
    envFrom:
      - secretRef:
          name: logstash-env
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    jvm: '-Xmx1G -Xms1G'
    replicas: 2
    resources:
      limits:
        cpu: '1'
        memory: 2Gi
      requests:
        cpu: 250m
        memory: 512Mi
    nodeSelector:
      project: cluster-dev
    tolerations:
    - effect: NoSchedule
      key: project
      operator: Equal
      value: cluster-dev
    podTemplate:
      spec:
        containers:
          - name: logstash
            command:
              - custom command
    podDisruptionBudget:
      maxUnavailable: 1
    antiAffinity:
      topologyKey: topology.kubernetes.io/zone
      type: hard
    additionalVolumes:
      - mountPath: /mnt/inputs
        name: inputs
        persistentVolumeClaim:
          claimName: pvc-logstash-inputs
    persistence:
      volumeClaim:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 10Gi
        storageClassName: openebs-hostpath
    ports:
      - containerPort: 5003
        hostPort: 5003
        name: beat
        protocol: TCP
```

**logstash-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: logstash-env
  namespace: cluster-dev
type: Opaque
data:
  ENV1: ++++++++
```


**pvc-logstash-input.yaml**:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-logstash-input
  namespace: cluster-dev
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
  storageClassName: nfs-client
```