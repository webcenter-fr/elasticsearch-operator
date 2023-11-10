# Kibana with managed cluster sample

In this sample, we will deploy an Elasticsearch cluster with single node on namespace `cluster-dev`. And we also deploy Kibana.

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
  clusterName: cluster-dev
  endpoint:
    ingress:
      enabled: true
      annotations:
        nginx.ingress.kubernetes.io/proxy-body-size: 512M
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
        nginx.ingress.kubernetes.io/ssl-redirect: 'true'
      host: elasticsearch-cluster-dev.domain.local
      secretRef:
        name: es-tls
  globalNodeGroup:
    config:
      elasticsearch.yml: |
        action.destructive_requires_name: true
        gateway.recover_after_time: 5m
        http.cors.allow-credentials: true
        http.cors.allow-headers: X-Requested-With,X-Auth-Token,Content-Type,
        Content-Length, Authorization
        http.cors.allow-origin: /.*/
        http.cors.enabled: true
        http.max_content_length: 500mb

        xpack.security.audit.enabled: true
        xpack.security.audit.logfile.events.exclude:
          - access_granted
        xpack.security.authc:
          anonymous:
            authz_exception: false
            roles: monitoring
            username: anonymous_user
        
        # Custom config
        cluster.routing.allocation.disk.watermark.flood_stage: 1gb
        cluster.routing.allocation.disk.watermark.high: 1gb
        cluster.routing.allocation.disk.watermark.low: 2gb
        gateway.expected_data_nodes: 1
        gateway.recover_after_data_nodes: 1
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
  nodeGroups:
    - name: all
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 20Gi
          storageClassName: local-path
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 4Gi
        requests:
          cpu: 250m
          memory: 4Gi
      roles:
        - master
        - data_hot
        - data_content
        - ingest
      waitClusterStatus: yellow
  setVMMaxMapCount: true
  version: 8.7.1
```

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
    managed:
      name: elasticsearch
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