# Deployment settings for Metricbeat


You can use the following setting to drive how to deploy Metricbeat:
- **replicas** (number / required): The number of instances
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it. Default is `soft` with the key `kubernetes.io/hostname`
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **resources** (object): The default resources for Metricbeat pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **podTemplate** (object): The pod template to merge with the Metricbeat pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- **initContainerResources** (object): The default resources for all init containers. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **additionalVolumes** (slice of object): Attach additional volume on Metricbeat pods. Default to `empty`
  - **name** (string / required): The volume name
  - **volumeMount** (string / required): The path to mount the volume inside the pod
  - **volumeSource** (object / required): The pod volume source. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)
- **persistence** (object): The persistent volume to use to store Elasticsearch data. Default is `emptyDir` (not peristent)
  - **volumeClaim** (object): Use it if you should to use PVC. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
  - **volume** (object): Use it if you should to use existing volume or hostPath. Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/storage/volumes/)


**metricbeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  name: metricbeat
  namespace: cluster-dev
spec:
  deployment:
    additionalVolumes:
      - mountPath: /usr/share/metricbeat/source-es-ca
        name: ca-source-elasticsearch
        secret:
          items:
            - key: ca.crt
              path: ca.crt
          secretName: elasticsearch-ca
    annotations:
      annotation1: my annotation
    labels:
      label1: my label
    env:
      - name: SOURCE_METRICBEAT_USERNAME
        value: remote_monitoring_user
      - name: SOURCE_METRICBEAT_PASSWORD
        valueFrom:
          secretKeyRef:
            key: remote_monitoring_user
            name: elasticsearch-credential
    envFrom:
      - secretRef:
          name: metricbeat-env
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    replicas: 1
    resources:
      limits:
        cpu: 300m
        memory: 200Mi
      requests:
        cpu: 100m
        memory: 100Mi
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
          - name: metricbeat
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
            storage: 10Gi
        storageClassName: openebs-hostpath
```

**metricbeat-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: metricbeat-env
  namespace: cluster-dev
type: Opaque
data:
  ENV1: ++++++++
```

**elasticsearch-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-ca
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**elasticsearch-credential-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-credential
  namespace: cluster-dev
type: Opaque
data:
  remote_monitoring_user: ++++++++
```