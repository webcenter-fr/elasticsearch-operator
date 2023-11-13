# Filebeat with managed cluster sample

In this sample, we will deploy one Elasticsearch cluster with single node on namespace `cluster-dev`. And we also deploy Filebeat to receive event from syslog input and send them to Elasticsearch.

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



**filebeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  labels:
    socle: cluster-dev
  name: filebeat
  namespace: cluster-dev
spec:
  config:
    filebeat.yml: |
      filebeat:
        shutdown_timeout: 5s

      logging:
        to_stderr: true
        level: info

      monitoring.enabled: false

      # Inputs
      filebeat.inputs:
        # Linux
        - type: syslog
          format: auto
          protocol.tcp:
            host: "0.0.0.0:5144"
          fields_under_root: true
          fields:
            event.dataset: "syslog_linux"
            event.module: "linux"
            service.type: "linux"
          tags: ["syslog"]
  deployment:
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    persistence:
      volumeClaim:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 5Gi
        storageClassName: openebs-hostpath
    ports:
      - containerPort: 5144
        name: linux
        protocol: TCP
    replicas: 1
    resources:
      limits:
        cpu: '1'
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 256Mi
    envFrom:
      - secretRef:
          name: filebeat-credential
  services:
    - name: syslog
      spec:
        ports:
          - name: linux
            nodePort: 30016
            port: 5144
            protocol: TCP
            targetPort: 5144
        type: NodePort
  version: 8.7.1
  elasticsearchRef:
    managed:
      name: elasticsearch
    secretRef:
      name: elasticsearch-credentials
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