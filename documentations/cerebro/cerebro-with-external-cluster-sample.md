# Cerebro with external cluster sample

In this sample, we will deploy Cerebro that target a remote Elasticsearch cluster.

**cerebro.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Cerebro
metadata:
  name: cerebro
  namespace: cluster-dev
spec:
  deployment:
    replicas: 1
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
      requests:
        cpu: 250m
        memory: 256Mi
  endpoint:
    ingress:
      enabled: true
      host: cerebro-cluster-dev.domain.local
      secretRef:
        name: cerebro-tls
  version: 0.9.4
  config:
    application.conf: |
      hosts = [
            {
                name = "elasticsearch-cluster-dev"
                host = "https://elasticsearch-cluster-dev"
            }
      ]
```