# Role

You can use the custom resource `Role` to manage the role inside Kibana.

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
  - **name** (string): The role name. Default it use the resource name.
  - **elasticsearch** (object): the Elasticsearch permissions. Default to empty.
    - **indices** (slice of object): The indice privileges. Default is empty.
      - **names** (slice of string / require): The list of indices. No default value.
      - **privileges** (slice of string / require): The list of privilege. No default value.
      - **fieldSecurity** (string): The field security of JSON format.
      - **query** (string): The query on JSON format.
    - **cluster** (slice of string): The list of cluster privilege. Default is empty.
    - **runAs** (slice of string): The list of privilege provided from user. Default is empty.
  - **kibana** (object): the Kibana permissions. Default to empty
    - **base** (slice of string): List of base privilege. Default is empty
    - **feature** (map of slice of string): The privilege for specific feature. Key is the feature name. Default is empty.
    - **spaces** (Slice of string): The list of user space to pply the privilege. Default is empty.
  - **transientMetadata** (object): The transient metadata. Default to empty
    - **enabled** (boolean /  require): Set to true to enable transient metadata
  - **metadata** (string): The metadata on JSON format. Default to empty

## Sample With managed Kibana

In this sample, we will create role on managed Kibana.

**role.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: Role
metadata:
  name: dev-read
  namespace: cluster-dev
spec:
  kibana:
    - base:
        - read
      spaces:
        - dev
  name: space_dev_read
  kibanaRef:
    managed:
      name: kibana
```

## Sample With external Kibana

In this sample, we will create role on external Kibana.

**role.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: Role
metadata:
  name: dev-read
  namespace: cluster-dev
spec:
  kibana:
    - base:
        - read
      spaces:
        - dev
  name: space_dev_read
  kibanaRef:
    external:
      address: https://kibana-dev.domain.com
    kibanaCASecretRef:
      name: custom-ca-kibana
    credentialSecretRef:
      name: kibana-credential
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