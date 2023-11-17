# Role mapping

You can use the custom resource `RoleMapping` to manage the role mapping inside Elasticsearch.

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
- **name** (string): The role mapping name. Default it use the resource name.
- **enabled** (boolean): Set to true to enable the role mapping. Default to `true`.
- **roles** (slice of string / require): The list of role. Default to empty.
- **rules** (string / require): The rules on JSON format.
- **metadata** (string): The metadata on JSON format. Default to empty

## Sample With managed Elasticsearch

In this sample, we will create role mapping on managed Elasticseach.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: RoleMapping
metadata:
  name: admins
  namespace: cluster-dev
spec:
  enabled: true
  roles:
    - superuser
    - admin
  rules: |
    {
      "any": [
          {
              "field": {
                "groups": "CN=ADMINS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          },
          {
              "field": {
                "groups": "CN=SUPPORTS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          }
      ]
    }
  elasticsearchRef:
    managed:
      name: elasticsearch
```

## Sample With external Elasticsearch

In this sample, we will create role mapping on external Elasticsearch.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: RoleMapping
metadata:
  name: admins
  namespace: cluster-dev
spec:
  enabled: true
  roles:
    - superuser
    - admin
  rules: |
    {
      "any": [
          {
              "field": {
                "groups": "CN=ADMINS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          },
          {
              "field": {
                "groups": "CN=SUPPORTS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          }
      ]
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