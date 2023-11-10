# Deployment settings for Kibana


You can use the following setting to drive how to deploy Kibana:
- **replicas** (number / required): The number of instances
- **antiAffinity** (object): The pod anti affinity to use. Hard or soft and the key to use to compute it. Default is `soft` with the key `kubernetes.io/hostname`
  - **type**: The anti affinity type. Default to `soft`
  - **topologyKey**: The toplogy key to use to compute anti affinity. Default to `kubernetes.io/hostname`.
- **resources** (object): The default resources for Kibana pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podDisruptionBudget** (object): The pod disruption budget to use. Default it allow to lost one pod. The selector is automatically set. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- **podTemplate** (object): The pod template to merge with the Kibana pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **waitClusterStatus** (string): Wait the cluster status provided for readiness check. Default to `green`.
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- **node** (string): It permit to set extra option on node process. Default to `empty`
- **initContainerResources** (object): The default resources for all init containers. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: kibana
  namespace: cluster-dev
spec:
  deployment:
    annotations:
      annotation1: my annotation
    labels:
      label1: my label
    env:
      - name: HTTP_PROXY
        value: 'http://squid.squid.svc:8080'
      - name: HTTPS_PROXY
        value: 'http://squid.squid.svc:8080'
      - name: NO_PROXY
        value: '.svc,localhost,127.0.0.1'
    envFrom:
      - secretRef:
          name: kibana-env
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    node: '--max-old-space-size=2048'
    replicas: 2
    resources:
      limits:
        cpu: '1'
        memory: 1Gi
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
          - name: kibana
            command:
              - custom command
    podDisruptionBudget:
      maxUnavailable: 1    
```

**kibana-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kibana-env
  namespace: cluster-dev
type: Opaque
data:
  ENV1: ++++++++
```
