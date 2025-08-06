# User

You can use the custom resource `User` to manage the user inside Elasticsearch.

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
- **enabled** (bool): Set false to disable account. Default to `true`.
- **username** (string): The user name. Default it use the resource name.
- **email** (string): The user email.
- **fullName** (string): The full name.
- **secretRef** (SecretKeySelector): The secret that store the credentials
  - **name** (string / required): The secret name
  - **key** (string / required): The secret key that store the credentials
- **passwordHash** (string): The password hashed. Not use it if you provide secret.
- **roles** (slice of string): The list of roles
- **isProtected** (bool): must be set when you manage protected account like kibana_system. Default to `false`.
- **autoGeneratePassword** (bool): set true if you should to auto generate password. Default to `false`
- **metadata** (map of string): The metadata. Default to empty

## Sample With managed Elasticsearch

In this sample, we will create role on managed Elasticseach.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: User
metadata:
  name: admin
  namespace: cluster-dev
spec:
  elasticsearchRef:
    managed:
      name: elasticsearch
  secretRef:
    name: credentials
    key: admin
  roles: ["superuser"]
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
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch-cluster-dev.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
  elasticsearchRef:
    managed:
      name: elasticsearch
  secretRef:
    name: credentials
    key: admin
  roles: ["superuser"]
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