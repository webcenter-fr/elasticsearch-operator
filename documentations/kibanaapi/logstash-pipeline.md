# Logstash pipeline

You can use the custom resource `LogstashPipeline` to manage the Logstash pipeline inside Kibana.

## Properties

You can use the following properties:
  - **kibanaRef** (object/ require): The kibana to connect on
    - **managed** (object): Use it if Kibana is managed by this operator
      - **name** (string / require): The kibana resource name
      - **namespace** (string): The namespace where Kibana is deployed. Default it use the current namespace.
    - **external** (object): Use it if Kibana is not managed by this operator
      - **address** (string / arequire): The URL to access on Kibana.
    - **kibanaCASecretRef** (object): The secret that store the custom CA to access on Kibana.. It need to have the key `ca.crt`
      - **name** (string / require): The secret name
    - **credentialSecretRef** (object): The secret that store the credentials to connect on Kibana. It need to contain the keys `username` and `password`.
      - **name** (string / require): The secret name
  - **name** (string): The pipeline ID. Default it use the resource name.
  - **description** (string): The pipeline description. Default to empty.
  - **pipeline** (string /  require): The pipeline spec. No default value
  - **settings** (string): The pipeline settings on JSON

## Sample With managed Kibana

In this sample, we will create Logstash pipeline on managed Kibana.

**logstash-pipeline.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: LogstashPipeline
metadata:
  name: logstashpipeline-sample
  namespace: cluster-dev
spec:
  kibanaRef:
    managed:
      name: kibana
  description: 'my logstash pipeline'
  settings: |
    {
      "queue.type": "persisted"
    }
  pipeline: |
    input { 
      stdin {} 
    } 
    output { 
      stdout {} 
    }
```

## Sample With external Kibana

In this sample, we will create Logstash pipeline on external Kibana.

**logstash-pipeline.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: LogstashPipeline
metadata:
  name: logstashpipeline-sample
  namespace: cluster-dev
spec:
  kibanaRef:
    external:
      address: https://kibana-dev.domain.com
    kibanaCASecretRef:
      name: custom-ca-kibana
    credentialSecretRef:
      name: kibana-credential
  description: 'my logstash pipeline'
  settings: |
    {
      "queue.type": "persisted"
    }
  pipeline: |
    input { 
      stdin {} 
    } 
    output { 
      stdout {} 
    }
```

**custom-ca-kibana-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca-kibana
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**kibana-credential-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kibana-credential
  namespace: cluster-dev
type: Opaque
data:
  username: ++++++++
  password: ++++++++
```