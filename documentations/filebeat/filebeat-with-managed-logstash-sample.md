# Filebeat with managed logstash
In this sample, we will deploy one Logstash on namespace `cluster-dev`. And we also deploy Filebeat to receive event from syslog input and send them to Logstash.

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
  logstashRef:
    managed:
      name: logstash
    logstashCASecretRef:
      name: filebeat-certificate
```

**filebeat-certificate.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: filebeat-certificate
  namespace: cluster-dev
type: Opaque
data:
  filebeat.crt: ++++++++
```