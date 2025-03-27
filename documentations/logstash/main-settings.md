# Main settings to deploy Logstash

You can use the following main setting to deploy Logstash:
- **image** (string): Logstash image to use. Default to `docker.elastic.co/logstash/logstash`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **pluginsList** (slice of string): The list of plugins to install on runtime (just before run Logstash). Use it for test purpose. For production, please build custom image to embedded your plugins. Default to `empty`.
- **config** (map of any): The Logstash config on YAML Format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on config folder. Each value is the file contend. It permit to set logstash.yml settings. Default is `empty`.
- **keystoreSecretRef** (object): The secrets to inject on keystore on runtime. Each keys / values is injected on Java Keystore. Default to `empty`.
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where Logstash connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
  - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`
    - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster.
    - **name** (string / require): The secret name
- **pipelines** (map of any): Each key is the file store on pipeline folder. Each value is the config on YAML format. It permit to set your pipeline spec. Default is `empty`.
- **patterns** (map of string): Each key is the file store on pattern folder. Each value is the file contend. It permit to set your custom grok patterns. Default is `empty`.

> The Logstash output is directly managed by your pipelines. So, the operator can't configure output for you.
> Moreover, you need to create a dedicated account for your Logstash Pipeline.

The operator will only exposed the following environement variable that you can use on your pipeline:
  - **ELASTICSEARCH_HOST**: The URL to connect on Elasticsearch
  - **ELASTICSEARCH_CA_PATH**: The CA path needed by elasticsearch output plugin.
  - **ELASTICSEARCH_USERNAME**: The username to used by Logtsash to connect on Elasticsearch
  - **ELASTICSEARCH_PASSWORD**: The password to used by Logstash to connect on Elasticsearch


**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  config:
    queue.type: persisted
    log.format: json
    dead_letter_queue.enable: true
    monitoring.enabled: false
    xpack.monitoring.enabled: false
    # Custom config
    pipeline.workers: 8
    queue.max_bytes: 20gb

    api.http.host: 0.0.0.0
  pipelines:
    log.yml: |
      input { stdin { } }
      output {
        stdout { codec => rubydebug }
      }
  patterns:
    postfix: |
      POSTFIX_QUEUEID [0-9A-F]{10,11}
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
  image: docker.elastic.co/logstash/logstash
  imagePullPolicy: Always
  imagePullSecrets:
    - name: my-pull-secret
  keystoreSecretRef:
    name: logstash-keystore
  pluginsList:
    - 'logstash-input-github'
  version: 8.7.1

```

**logstash-keystore-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: logstash-keystore
  namespace: cluster-dev
type: Opaque
data:
  key1: ++++++++
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