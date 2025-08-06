# Role

You can use the custom resource `Role` to manage the role inside Elasticsearch.

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
- **name** (string): The role name. Default it use the resource name.
- **indices** (slice of object): The indice privileges. Default is empty.
  - **names** (slice of string / require): The list of indices. No default value.
  - **privileges** (slice of string / require): The list of privilege. No default value.
  - **fieldSecurity** (string): The field security of JSON format.
  - **query** (string): The query on JSON format.
  - **allowRestrictedIndices** (boolean): set to true to allow manage restricted indice (system indice). Default to `false`.
- **cluster** (slice of string): The list of cluster privilege. Default is empty.
- **runAs** (slice of string): The list of privilege provided from user. Default is empty.
- **applications** (slice of object): the list of application privilege. Default is empty
  - **application** (string / require): The application name. No default value.
  - **privileges** (slice of string): The list of privilege. Default to empty.
  - **resources** (slice of string): The list of resources. Default to empty.
- **global** (string): The global privilege on JSON format/
- **transientMetadata** (string): The transient metadata on JSON format. Default to empty
- **metadata** (map of string): The metadata. Default to empty

## Sample With managed Elasticsearch

In this sample, we will create role on managed Elasticseach.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: Role
metadata:
  name: admin
  namespace: cluster-dev
spec:
  cluster:
    - all
  indices:
    - allowRestrictedIndices: true
      names:
        - '*'
      privileges:
        - all
  elasticsearchRef:
    managed:
      name: elasticsearch
```

## Sample With external Elasticsearch

In this sample, we will create role on external Elasticsearch.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: Role
metadata:
  name: admin
  namespace: cluster-dev
spec:
  cluster:
    - all
  indices:
    - allowRestrictedIndices: true
      names:
        - '*'
      privileges:
        - all
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