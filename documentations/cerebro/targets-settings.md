# Targets settings for Cerebro

You can use the `Host` CRD to define the Cerebro targets.

> The `Host` resource must be created on same namespace where Elasticsearch to enroll is deployed on.

You can use the following setting to declare target:
- **cerebroRef** (object / require): the Cerebro where to define new target
  - **name** (string / require): the cerebro resource name
  - **namespace** (string): The namespace where Cerebro is deployed on. Default is the same namespace where Host is created.
- **elasticsearchRef** (string /require): The Elasticsearch target
  - **managed** (object): Use it if target Elasticsearch is managed by this operator
    - **name** (string / require): The Elasticsearch resource name
  - **external** (object): Use it if target Elasticsearch is not managed by this operator
    - **name** (string / require): The cluster name to display on Cerebro
    - **address** (string / require): The URL to connect on Elasticsearch cluster

**host.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Host
metadata:
  name: cerebro
  namespace: cluster-dev
spec:
  cerebroRef:
    name: cerebro
  elasticsearchRef:
    managed:
      name: elasticsearch
    external:
      name: cluster-dev
      address: https://elasticsearch-cluster-dev.domain.local
```