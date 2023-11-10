# Main settings to deploy Logstash

You can use the following main setting to deploy Logstash:
- **image** (string): Kibana image to use. Default to `docker.elastic.co/logstash/logstash`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **pluginsList** (slice of string): The list of plugins to install on runtime (just before run Logstash). Use it for test purpose. For production, please build custom image to embedded your plugins. Default to `empty`.
- **config** (map of string): Each key is the file store on config folder. Each value is the file contend. It permit to set logstash.yml settings. Default is `empty`.
- **keystoreSecretRef** (object): The secrets to inject on keystore on runtime. Each keys / values is injected on Java Keystore. Default to `empty`.
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where kibana connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
    - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`
      - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster.
    - **name** (string / require): The secret name
- **pipeline** (map of string): Each key is the file store on pipeline folder. Each value is the file contend. It permit to set your pipeline spec. Default is `empty`.
- **pattern** (map of string): Each key is the file store on pattern folder. Each value is the file contend. It permit to set your custom grok patterns. Default is `empty`.
- **ingresses** (slice of Object): It's permit to create custom ingresses if needed by input or to access on Logstash API. It will set automatically the service needed by ingress.
  - **name** (string / require): the ingress name
  - **spec** (object / require): the ingress spec. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/services-networking/ingress/).
  - **labels** (map of string): The list of labels to add on ingress. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on ingress. Default to `empty`
  - **containerProtocol** (string): the protocol to set when create service consumed by ingress.`udp` or `tcp`
  - **containerPort** (number /require): the port to set when create service consumed by ingress
- **services** (slice of services): it's permit to create custom services if needed by input
  - **name** (string / require): the service name
  - **spec** (object / require): the service spec.  Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/services-networking/service/)
  - **labels** (map of string): The list of labels to add on service. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on service. Default to `empty`


**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
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

      api.http.host: 0.0.0.0
  pipeline:
    log.yml: |
      input { stdin { } }
      output {
        stdout { codec => rubydebug }
      }
  pattern:
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
  keystoreSecretRef:
    name: logstash-keystore
  pluginsList:
    - 'logstash-input-github'
  services:
    - name: beat
      labels:
        label1: my label
      annotations:
        annotation1: my annotation
      spec:
        ports:
          - name: beats
            port: 5003
            protocol: TCP
            targetPort: 5003
        type: ClusterIP
  ingresses:
    - name: api
      spec:
        rules:
          - host: logstash-api-dev.domain.local
            http:
              paths:
                - backend:
                    service:
                      name: logstash-api
                      port:
                        number: 9600
                  path: /
                  pathType: Prefix
        tls:
          - hosts:
              - logstash-api-dev.domain.local
            secretName: ls-tls
      labels:
        label1: my label
      annotations:
        annotation1: my annotation
      containerProtocol: tcp
      containerPort: 9600
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