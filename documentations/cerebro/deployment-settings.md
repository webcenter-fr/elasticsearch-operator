# Deployment settings for Cerebro


You can use the following setting to drive how to deploy Cerebro:
- **replicas** (number / required): The number of instances
- **resources** (object): The default resources for Cerebro pods. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **podTemplate** (object): The pod template to merge with the Cerebro pod template. Default is empty. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/workloads/pods/)
- **labels** (map of string): The labels to merge on pod. Default to `empty`.
- **annotations** (map of string): The annotations to merge on pod. Default to `empty`.
- **env** (slice of object): The environment variable to inject on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **envFrom** (slice of object): The secret or configMap to inject as environement variable on Elasticsearch pod. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
- **nodeSelector** (map of string): The node slector constraint. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- **tolerations** (slice of object): The toleration to schedule pod on nodes. Default to `empty`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- **node** (string): It permit to set extra option on node process. Default to `empty`

**cerebro.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Cerebro
metadata:
  name: cerebro
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
          name: cerebro-env
    node: '--max-old-space-size=2048'
    replicas: 1
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
          - name: cerebro
            command:
              - custom command
```

**cerebro-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cerebro-env
  namespace: cluster-dev
type: Opaque
data:
  ENV1: ++++++++
```
