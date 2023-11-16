# Cerebro with managed cluster sample

In this sample, we will deploy an Elasticsearch cluster with single node on namespace `cluster-dev`. And we also deploy Cerebro. To finish, we create a `Host` resource to enroll our Elasticsearch cluster as target on Cerebro.

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
```

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
```