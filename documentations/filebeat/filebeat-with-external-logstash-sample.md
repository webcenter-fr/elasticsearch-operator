# Filebeat with external logstash
In this sample, we will deploy Filebeat to receive event from syslog input and send them to external Logstash (not managed by operator).

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
    external:
      addresses:
        - logstash-cluster-dev.domain.local:5003
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