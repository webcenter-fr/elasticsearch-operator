# Logstash with external cluster sample

In this sample, we will deploy Kibana that use external Elasticsearch cluster (or cluster not manafed by this operator).

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
    external:
      addresses:
        - https://elasticsearch-cluster-dev.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
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