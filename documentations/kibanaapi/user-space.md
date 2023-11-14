# User space

You can use the custom resource `UserSpace` to manage the user space inside Kibana.

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
  - **id** (string): The user space ID. Default it use the resource name.
  - **name** (string / require): The user space name. No default value.
  - **description** (string): The user space description. Default to empty.
  - **disabledFeatures** (slice of string): The features to disable on user space. Default is empty.
  - **initials** (string): the user space initials. Default is auto generated by Kibana.
  - **color** (string): the user space color. Default is auto generated by Kibana
  - **userSpaceCopies** (slice of object). It permit to copy on or more object from other user space. It usefull to init the data-view or settings. Default is empty
    - **originUserSpace** (string / require): the user space name from copy objects. No default value
    - **includeReferences** (boolean): set to true to copy all references. Default to `true`.
    - **overwrite** (boolean): set to true to overwrite existing objects. Default to `true`.
    - **createNewCopies** (boolean): set to true to create new copy of objects. Default to `false`.
    - **forceUpdate** (boolean): set to true to force to sync objects each time the operator reconcile. Default to `false`.
    - **objects** (slice of objects / require): The list of objects to copy. No default value.
      - **type** (string / require): the object type. No default value
      - **id** (string / require): The object ID. No default value



## Sample With managed Kibana

In this sample, we will create user space on managed Kibana.

**user-space.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: UserSpace
metadata:
  name: user-space
  namespace: cluster-dev
spec:
  name: dev
  id: dev
  description: Space for dev team
  disabledFeatures:
    - advancedSettings
    - graph
    - monitoring
    - ml
    - apm
    - infrastructure
    - siem
    - uptime
  userSpaceCopies:
    - objects:
        - id: logs-*
          type: index-pattern
      originUserSpace: default
  kibanaRef:
    managed:
      name: kibana
```

## Sample With external Kibana

In this sample, we will create user space on external Kibana.

**user-space.yml**:
```yaml
apiVersion: kibanaapi.k8s.webcenter.fr/v1
kind: UserSpace
metadata:
  name: user-space
  namespace: cluster-dev
spec:
  name: dev
  id: dev
  description: Space for dev team
  disabledFeatures:
    - advancedSettings
    - graph
    - monitoring
    - ml
    - apm
    - infrastructure
    - siem
    - uptime
  userSpaceCopies:
    - objects:
        - id: logs-*
          type: index-pattern
      originUserSpace: default
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