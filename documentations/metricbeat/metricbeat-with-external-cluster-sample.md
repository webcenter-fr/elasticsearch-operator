# Metricbeat with external cluster sample

In this sample, we will deploy Metricbeat that consume metric from external Elastic cluster to external monitoring cluster (not managed by operator).

**metricbeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  labels:
    socle: cluster-dev
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
          secretName: elasticsearch-source-ca
    env:
      - name: SOURCE_METRICBEAT_USERNAME
        value: remote_monitoring_user
      - name: SOURCE_METRICBEAT_PASSWORD
        valueFrom:
          secretKeyRef:
            key: remote_monitoring_user
            name: elasticsearch-source-credential
    replicas: 1
    resources:
      limits:
        cpu: 300m
        memory: 200Mi
      requests:
        cpu: 100m
        memory: 100Mi
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch-cluster-monitoring.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
  module:
    elasticsearch-xpack.yml:
      - module: elasticsearch
        xpack.enabled: true
        username: '${SOURCE_METRICBEAT_USERNAME}'
        password: '${SOURCE_METRICBEAT_PASSWORD}'
        ssl:
          enable: true
          certificate_authorities: '/usr/share/metricbeat/source-es-ca/ca.crt'
          verification_mode: full
        scope: cluster
        period: 10s
        hosts: https://elasticsearch-cluster-dev.domain.local
  version: 8.7.1

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

**elasticsearch-source-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-source-ca
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**elasticsearch-source-credential-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-source-credential
  namespace: cluster-dev
type: Opaque
data:
  remote_monitoring_user: ++++++++
```