# Deployment settings for Filebeat


You can use the following setting to drive how to deploy Filebeat:
- **replicas** (number / required): The number of instances
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it. Default is `soft` with the key `kubernetes.io/hostname`
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **resources** (object): The default resources for Filebeat pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **podTemplate** (object): The pod template to merge with the Filebeat pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- **initContainerResources** (object): The default resources for all init containers. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **additionalVolumes** (slice of object): Attach additional volume on Filebeat pods. Default to `empty`
  - **name** (string / required): The volume name
  - **volumeMount** (string / required): The path to mount the volume inside the pod
  - **volumeSource** (object / required): The pod volume source. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **persistence** (object): The persistent volume to use to store Elasticsearch data. Default is `emptyDir` (not peristent)
  - **volumeClaim** (object): Use it if you should to use PVC. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).You can set here `labels` and `annotations`.
  - **volume** (object): Use it if you should to use existing volume or hostPath. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **ports** (slice of object): It permit to set container ports on Pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)


**filebeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
  namespace: cluster-dev
spec:
  deployment:
    additionalVolumes:
      - mountPath: /usr/share/filebeat/certs
        name: logstash-certificates
        secret:
          secretName: logstash-certificates
    annotations:
      annotation1: my annotation
    labels:
      label1: my label
    env:
      - name: ENV1
        value: test
    envFrom:
      - secretRef:
          name: filebeat-env
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
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
          - name: filebeat
            command:
              - custom command
    podDisruptionBudget:
      maxUnavailable: 1
    antiAffinity:
      topologyKey: topology.kubernetes.io/zone
      type: hard
    persistence:
      volumeClaim:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 5Gi
        storageClassName: openebs-hostpath
    ports:
      - containerPort: 5144
        name: linux
        protocol: TCP
    replicas: 1
    resources:
      limits:
        cpu: '1'
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 256Mi
```

**filebeat-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: filebeat-env
  namespace: cluster-dev
type: Opaque
data:
  ENV1: ++++++++
```

**logstash-certificates-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: logstash-certificates
  namespace: cluster-dev
type: Opaque
data:
  filebeat.crt: ++++++++
```