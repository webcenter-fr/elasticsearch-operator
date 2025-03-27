# Main settings to deploy Filebeat

You can use the following main setting to deploy Filebeat:
- **image** (string): Metricbeat image to use. Default to `docker.elastic.co/beats/filebeat`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **config** (map of any): The config on Yaml format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on filebeat folder. Each value is the file contend. It permit to set filebeat.yml settings.
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where Metricbeat connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster 
  - **secretRef** (object / require): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`.
    - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). The secret ref that store the CA certificate to connect on Elasticsearch. It need to contain the keys `ca.crt`.
    - **name** (string / require): The secret name
- **logstashRef** (object): The Logstash instance ref
  - **managed** (object): Use it if Logstash is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetService** (string): the target service that expose the beat 
    - **port** (number / require): The port to connect on
  - **external** (object): Use it if Logstash is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on Logstash
  - **logstashCASecretRef** (object). It's the secret that store custom CA to connect on Logstash instance. If you set managed Logstash, and you have enabled internal PKI on Logstash, it will inject automatically the CA certificate on POD and configure the Logstash output with it.
    - **name** (string / require): The secret name
- **modules** (map of any): Each key is the file store on modules.d folder. Each value is the YAML contend. It permit to enable and configure modules. Default is `empty`.


**filebeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
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
    # Logstash settings
    output.logstash:
      timeout: 15
      ssl:
        enable: true
        certificate_authorities:
          - /usr/share/filebeat/certs/filebeat.crt
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
  logstashRef:
    managed:
      name: logstash-log
      port: 5003
    external:
      addresses:
        - logstash-cluster-dev.domain.local:5003
    logstashCASecretRef:
      name: logstash-custom-ca
  elasticsearchRef:
    managed:
      name: elasticsearch
      namespace: cluster-dev
      targetNodeGroup: client
    external:
      addresses:
        - https://cluster-dev.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: elasticsearch-custom-ca
  image: docker.elastic.co/beats/filebeat
  imagePullPolicy: Always
  imagePullSecrets:
    - name: my-pull-secret
  version: 8.7.1
  module:
    iptables.yml:
      - log:
          enabled: true
          var.paths: ["/var/log/iptables.log"]
          var.input: "file"
```

**my-pull-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-pull-secret
  namespace: cluster-dev
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: UmVhbGx5IHJlYWxseSByZWVlZWVlZWVlZWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWxsbGxsbGxsbGxsbGxsbGxsbGxsbGxsbGxsbGxsbGx5eXl5eXl5eXl5eXl5eXl5eXl5eSBsbGxsbGxsbGxsbGxsbG9vb29vb29vb29vb29vb29vb29vb29vb29vb25ubm5ubm5ubm5ubm5ubm5ubm5ubm5ubmdnZ2dnZ2dnZ2dnZ2dnZ2dnZ2cgYXV0aCBrZXlzCg==
```

**elasticsearch-custom-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-custom-ca
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**elasticsearch-credentials-secret.yaml**:
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

**logstash-custom-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: logstash-custom-ca
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```