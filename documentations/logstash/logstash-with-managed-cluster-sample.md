# Logstash with managed cluster sample

In this sample, we will deploy an Elasticsearch cluster with single node on namespace `cluster-dev`. And we also deploy Logstash.

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


**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  labels:
    socle: cluster-dev
  name: logstash
  namespace: cluster-dev
spec:
  config:
    logstash.yml: |
      queue.type: persisted
      log.format: json
      dead_letter_queue.enable: true
      monitoring.enabled: false
      xpack.monitoring.enabled: false

      # Custom config
      pipeline.workers: 8
      queue.max_bytes: 20gb
  pipeline:
    log.yml: |
      input { stdin { } }
      output {
        stdout { codec => rubydebug }
      }
  deployment:
    additionalVolumes:
      - mountPath: /usr/share/logstash/certs
        name: logstash-certificates
        secret:
          secretName: logstash-certificates
    env:
      - name: http_proxy
        value: 'http://squid.squid.svc:8080'
      - name: https_proxy
        value: 'http://squid.squid.svc:8080'
      - name: no_proxy
        value: .svc
    envFrom:
      - secretRef:
          name: logstash-credentials
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    jvm: '-Xmx4G -Xms2G'
    persistence:
      volumeClaim:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 20Gi
        storageClassName: openebs-hostpath
    ports:
      - containerPort: 5003
        hostPort: 5003
        name: beat
        protocol: TCP
    replicas: 4
    resources:
      limits:
        cpu: '4'
        memory: 6Gi
      requests:
        cpu: '1'
        memory: 2Gi
  elasticsearchRef:
    managed:
      name: elasticsearch
  services:
    - name: beat
      spec:
        ports:
          - name: beats
            port: 5003
            protocol: TCP
            targetPort: 5003
        type: ClusterIP
  version: 8.7.1

```