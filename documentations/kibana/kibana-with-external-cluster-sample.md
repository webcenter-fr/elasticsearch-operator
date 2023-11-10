# Kibana with external cluster sample

In this sample, we will deploy Kibana that use external Elasticsearch cluster (or cluster not manafed by this operator).


**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  labels:
    socle: cluster-dev
  name: kibana
  namespace: cluster-dev
spec:
  config:
    kibana.yml: |
      elasticsearch.requestTimeout: 300000
      unifiedSearch.autocomplete.valueSuggestions.timeout: 3000
      xpack.reporting.roles.enabled: false
      monitoring.kibana.collection.enabled: false
      monitoring.ui.enabled: false
  deployment:
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    node: '--max-old-space-size=2048'
    replicas: 1
    resources:
      limits:
        cpu: '1'
        memory: 1Gi
      requests:
        cpu: 250m
        memory: 512Mi
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch-cluster-dev.domain.local
      secretRef:
        name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
  endpoint:
    ingress:
      annotations:
        nginx.ingress.kubernetes.io/proxy-body-size: 4G
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
        nginx.ingress.kubernetes.io/ssl-redirect: 'true'
      enabled: true
      host: kibana-dev.domain.local
      secretRef:
        name: kb-tls
  keystoreSecretRef:
    name: kibana-keystore
  version: 8.7.1
```

**kibana-keystore-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kibana-keystore
  namespace: cluster-dev
type: Opaque
data:
  xpack.encryptedSavedObjects.encryptionKey: ++++++++
  xpack.reporting.encryptionKey: ++++++++
  xpack.security.encryptionKey: ++++++++
```

**custom-ca-elasticsearch-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca-elasticsearch
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**elasticsearch-credentials.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-credentials
  namespace: cluster-dev
type: Opaque
data:
  username: ++++++++
  password: ++++++++
```