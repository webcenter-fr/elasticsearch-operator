# Index template
You can use the custom resource `IndexTemplate` to manage the index template inside Elasticsearch.

## Properties

You can use the following properties:
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where operator connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
  - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`. It only used for external Elasticsearch.
    - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster.
    - **name** (string / require): The secret name
- **name** (string): The index template name. Default it use the resource name.
- **indexPatterns** (slice of string): The list of index pattern to apply this template. Default to empty
- **composedOf** (slice of string): The list of component templates. Default to empty
- **priority** (number): The priority to apply this template. Default to empty
- **version** (number): The index template version. Default to empty.
- **template** (object): The template specification. Default to empty.
  - **settings** (string): The template setting in JSON string format. Default to empty.
  - **mappings** (string): The template mapping in JSON string format. Default to empty.
  - **aliases** (string): The template alias in JSON string format. Default to empty.
- **meta** (string): The extended info as JSON string. Default to empty.
- **allowAutoCreate** (boolean): It permit to allow auto create index. Default to false.
- **rawTemplate** (string): The template in raw format (JSON string format).  You can use it instead to set all properties.

## Sample With managed Elasticsearch

In this sample, we will create index template on managed Elasticseach.

**template.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: IndexTemplate
metadata:
  name: test
  namespace: cluster-dev
spec:
  indexPatterns: ["te*", "bar*"]
  priority: 500
  composedOf: ["component_template1", "runtime_component_template"]
  version: 3
  meta: |
    {
        "description": "my custom"
    }
  template:
    settings: |
      {
        "number_of_shards": 1
      },
    mappings: |
      {
        "_source": {
          "enabled": true
        },
        "properties": {
          "host_name": {
            "type": "keyword"
          },
          "created_at": {
            "type": "date",
            "format": "EEE MMM dd HH:mm:ss Z yyyy"
          }
        }
      }
    aliases: |
      {
        "mydata": { }
      }
  elasticsearchRef:
    managed:
      name: elasticsearch
```

Or if you prefer in raw format

**component.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: IndexTemplate
metadata:
  name: test
  namespace: cluster-dev
spec:
  rawTemplate: |
    {
      "index_patterns": ["te*", "bar*"],
      "template": {
        "settings": {
          "number_of_shards": 1
        },
        "mappings": {
          "_source": {
            "enabled": true
          },
          "properties": {
            "host_name": {
              "type": "keyword"
            },
            "created_at": {
              "type": "date",
              "format": "EEE MMM dd HH:mm:ss Z yyyy"
            }
          }
        },
        "aliases": {
          "mydata": { }
        }
      },
      "priority": 500,
      "composed_of": ["component_template1", "runtime_component_template"], 
      "version": 3,
      "_meta": {
        "description": "my custom"
      }
    }
  elasticsearchRef:
    managed:
      name: elasticsearch
```

## Sample With external Elasticsearch

In this sample, we will create index template on external Elasticsearch.

**component.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: IndexTemplate
metadata:
  name: test
  namespace: cluster-dev
spec:
  indexPatterns: ["te*", "bar*"]
  priority: 500
  composedOf: ["component_template1", "runtime_component_template"]
  version: 3
  meta: |
    {
        "description": "my custom"
    }
  template:
    settings: |
      {
        "number_of_shards": 1
      },
    mappings: |
      {
        "_source": {
          "enabled": true
        },
        "properties": {
          "host_name": {
            "type": "keyword"
          },
          "created_at": {
            "type": "date",
            "format": "EEE MMM dd HH:mm:ss Z yyyy"
          }
        }
      }
    aliases: |
      {
        "mydata": { }
      }
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch-cluster-dev.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
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