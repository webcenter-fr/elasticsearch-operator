# Main settings to deploy Kibana

You can use the following main setting to deploy Kibana:
- **image** (string): Kibana image to use. Default to `docker.elastic.co/kibana/kibana`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **pluginsList** (slice of string): The list of plugins to install on runtime (just before run Kibana). Use it for test purpose. For production, please build custom image to embedded your plugins. Default to `empty`.
- **config** (map of any): The Kibana config on YAML format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on config folder. Each value is the file contend. It permit to set kibana.yml settings. Default is `empty`.
- **keystoreSecretRef** (object): The secrets to inject on keystore on runtime. Each keys / values is injected on Java Keystore. Default to `empty`.
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where kibana connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
  - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`. It only used for external Elasticsearch.
    - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster.
    - **name** (string / require): The secret name

**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: kibana
  namespace: cluster-dev
spec:
  config:
    elasticsearch.requestTimeout: 300000
    unifiedSearch.autocomplete.valueSuggestions.timeout: 3000
    xpack.reporting.roles.enabled: false
    monitoring.ui.enabled: false
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
  keystoreSecretRef:
    name: kibana-keystore
  version: 8.7.1
  image: docker.elastic.co/kibana/kibana
  imagePullPolicy: IfNotPresent
  imagePullSecrets:
    - name: my-pull-secret
  pluginsList:
    - 'https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/8.10.0/kibana-prometheus-exporter-8.10.0.zip'
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