# Metricbeat with managed cluster sample

In this sample, we will deploy 2 Elasticsearch cluster with single node on namespace `cluster-dev`. One cluster for monitoring and one cluster for application. And we also deploy Metricbeat to collect metric from application cluster to send them on monitoring cluster.

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

**elasticsearch-monitoring.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  labels:
    socle: cluster-dev
  name: elasticsearch-monitoring
  namespace: cluster-dev
spec:
  clusterName: cluster-monitoring
  endpoint:
    ingress:
      enabled: true
      annotations:
        nginx.ingress.kubernetes.io/proxy-body-size: 512M
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
        nginx.ingress.kubernetes.io/ssl-redirect: 'true'
      host: elasticsearch-cluster-monitoring.domain.local
      secretRef:
        name: es-tls
  globalNodeGroup:
    config:
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
          secretName: elasticsearch-tls-api-es
    env:
      - name: SOURCE_METRICBEAT_USERNAME
        value: remote_monitoring_user
      - name: SOURCE_METRICBEAT_PASSWORD
        valueFrom:
          secretKeyRef:
            key: remote_monitoring_user
            name: elasticsearch-credential-es
    replicas: 1
    resources:
      limits:
        cpu: 300m
        memory: 200Mi
      requests:
        cpu: 100m
        memory: 100Mi
  elasticsearchRef:
    managed:
      name: elasticsearch-monitoring
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
        hosts: https://elasticsearch-es.cluster-dev.svc:9200
  version: 8.7.1
```

> We consule the secret created by operator when deploy `elasticsearch` cluster` to connect on.