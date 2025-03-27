# Main settings to deploy Metricbeat

You can use the following main setting to deploy Metricbeat:
- **image** (string): Metricbeat image to use. Default to `docker.elastic.co/beats/metricbeat`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **config** (map of any): The config or Metricbeat on YAML format. Default is `empty`.
- **extraConfigs** (map of string): Each key is the file store on metricbeat folder. Each value is the file contend. It permit to set metricbeat.yml settings. Default is `empty`.
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where Metricbeat connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
  - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`. It only used for external Elasticsearch.
      - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster. The key must be `ca.crt`
    - **name** (string / require): The secret name
- **module** (map of any): Each key is the file store on modules.d folder. Each value is the config on YAML format. It permit to enable and configure modules. Default is `empty`.


**metricbeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  name: metricbeat
  namespace: cluster-dev
spec:
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
  image: docker.elastic.co/beats/metricbeat
  imagePullPolicy: Always
  imagePullSecrets:
    - name: my-pull-secret
  config:
    tags: ["service-X", "web-tier"]
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
        hosts: https://elasticsearchcluster-dev.svc:9200
  version: 8.7.1

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