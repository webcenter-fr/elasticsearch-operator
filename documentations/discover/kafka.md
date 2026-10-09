# Kafka Discover

The Kafka Discover resource provides a unified way to manage Kafka connection information for Filebeat and Logstash components. It generates and maintains secrets containing all necessary information to connect on Kafka clusters: CA certificates, keystores, truststores, credentials and bootstrap servers.

## Overview

The Kafka Discover resource automatically:
- Generates connection secrets with file-based (certificates, keystores) and environment variable (bootstrap servers, passwords) formats
- Manages SSL/TLS certificates and PKCS#12 keystores/truststores for secure connections
- Supports both managed Kafka clusters (Strimzi operator) and external Kafka clusters
- Watches for changes and updates dependent resources automatically (workloads are restarted when the secrets change)

> Kafka managed by Strimzi requires the Strimzi operator and its CRDs to be installed on the cluster. If not installed, only `kafkaRef.external` can be used.

## Resource Definition

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: kafka-sample
  namespace: cluster-dev
spec:
  # Target secrets configuration
  targetSecretRef:
    name: kafka-sample-file        # Optional: defaults to {name}-file
  targetSecretEnvRef:
    name: kafka-sample-env         # Optional: defaults to {name}-env
  name: KAFKA_SAMPLE               # Optional: suffix for env vars / mount dir, defaults to resource name

  # Kafka connection reference
  kafkaRef:
    # For managed Kafka (Strimzi operator)
    managed:
      name: my-kafka-cluster
      namespace: kafka-namespace   # Optional: defaults to same namespace
      targetListener: tls          # Optional: listener name
      userRef:
        name: my-kafka-user        # Optional: reference to a KafkaUser resource

    # OR for external Kafka
    external:
      addresses:
        - kafka-broker1.example.com:9092
        - kafka-broker2.example.com:9092
      userSecretRef:
        name: kafka-ssl-user       # Optional: for SSL client authentication

    # Optional: Custom CA certificates
    kafkaCASecretRef:
      name: custom-kafka-ca
```

## Configuration Options

### Target Secrets

| Field | Description | Default |
|-------|-------------|---------|
| `targetSecretRef.name` | Secret name for file-based information (certificates, private keys, keystores, truststores) | `{resource-name}-file` |
| `targetSecretEnvRef.name` | Secret name for environment variables | `{resource-name}-env` |
| `name` | Logical name used as suffix for environment variables and as directory name to mount the file secret on pods | Resource name |

### Managed Kafka Reference

For Kafka clusters managed by the Strimzi operator:

| Field | Description | Required |
|-------|-------------|----------|
| `managed.name` | Name of the Strimzi `Kafka` resource | Yes |
| `managed.namespace` | Namespace of the Strimzi `Kafka` resource | No (defaults to same namespace) |
| `managed.targetListener` | Name of the listener to use. Default: the first `internal` or `clusterip` listener, else the first listener | No |
| `managed.userRef.name` | Name of the Strimzi `KafkaUser` resource used for client authentication | No |

The referenced `KafkaUser` status must expose its secret. The user certificate (`user.crt`) and private key (`user.key`) are then copied to the file secret and converted to a PKCS#12 keystore.

### External Kafka Reference

For Kafka clusters not managed by Strimzi:

| Field | Description | Required |
|-------|-------------|----------|
| `external.addresses` | List of Kafka bootstrap addresses (`host:port`) | Yes |
| `external.userSecretRef.name` | Secret containing client SSL certificate and key | No |

The user secret may contain:
- `user.crt`: Client certificate
- `user.key`: Client private key
- `ca.crt`: CA certificate used to sign the client certificate (optional)

### CA Certificate Reference

| Field | Description |
|-------|-------------|
| `kafkaCASecretRef.name` | Secret containing custom CA certificates. Every entry ending by `.crt` or `.pem` is added to the generated CA bundle |

If Kafka is managed by Strimzi with the internal PKI, the cluster CA (`<kafka-name>-cluster-ca-cert`) is used automatically.

## Authentication Types

The authentication type is detected from the selected Strimzi listener and exposed through the `KAFKA_AUTH_TYPE_{SUFFIX}` environment variable. Supported Strimzi authentication types are `tls`, `scram-sha-512`, `oauth` and `plain`.

When OAuth is configured, the following variables are also generated:

| Variable | Description |
|----------|-------------|
| `KAFKA_OAUTH_CLIENT_ID_{SUFFIX}` | OAuth client id |
| `KAFKA_OAUTH_TOKEN_ENDPOINT_URI_{SUFFIX}` | OAuth token endpoint URI |
| `KAFKA_OAUTH_VALID_ISSUER_URI_{SUFFIX}` | OAuth valid issuer URI |
| `KAFKA_OAUTH_JWKS_ENDPOINT_URI_{SUFFIX}` | OAuth JWKS endpoint URI |
| `KAFKA_OAUTH_USERNAME_CLAIM_{SUFFIX}` | OAuth username claim |

## Generated Secrets

The Kafka Discover resource generates two secrets, owned by the `Kafka` resource.

### File Secret (`{name}-file`)

Contains certificates and keystores as files:

| Key | Description |
|-----|-------------|
| `ca.crt` | Concatenation of all CA certificates (Strimzi cluster CA and custom CA) |
| `ca.p12` | PKCS#12 truststore generated from the CA certificates |
| `user.crt` | User certificate, when client authentication is configured |
| `user.key` | User private key, when client authentication is configured |
| `user.p12` | PKCS#12 keystore generated from the user certificate and private key |

### Environment Variable Secret (`{name}-env`)

Contains connection information as environment variables. The suffix (`{SUFFIX}`) is the logical name (`spec.name` or the resource name) uppercased with `-` replaced by `_`:

| Variable | Description |
|----------|-------------|
| `KAFKA_BOOTSTRAP_SERVERS_{SUFFIX}` | Comma-separated list of Kafka bootstrap servers |
| `KAFKA_AUTH_TYPE_{SUFFIX}` | Authentication type of the selected listener (managed Kafka only) |
| `KAFKA_OAUTH_*_{SUFFIX}` | OAuth settings (managed Kafka with OAuth authentication only) |
| `KAFKA_CA_TRUSTSTORE_PASSWORD_{SUFFIX}` | Generated password of the CA truststore (`ca.p12`) |
| `KAFKA_USER_PASSWORD_{SUFFIX}` | Generated password of the user keystore (`user.p12`), when a user certificate is discovered |

For example, with a resource named `kafka-sample` (and no `spec.name`), the suffix is `KAFKA_SAMPLE` and the bootstrap variable is `KAFKA_BOOTSTRAP_SERVERS_KAFKA_SAMPLE`.

### Secret Metadata

Both secrets carry the following labels and annotations, used by workloads to select and mount them:

```yaml
labels:
  discoverName: kafka-sample
  discover.k8s.webcenter.fr: "true"
  kafka.discover.k8s.webcenter.fr: "true"
annotations:
  discover.k8s.webcenter.fr: "true"
  discover.k8s.webcenter.fr/type: "env"   # or "file"
  discover.k8s.webcenter.fr/mountPath: kafka-sample
  kafka.discover.k8s.webcenter.fr: "true"
```

## Certificate File Mounting

The file secret of each referenced Discover resource is mounted into the workload pods at:

`/usr/share/{application}/discover/{internal-name}/`

where:
- `{application}` is `filebeat`, `logstash` or `metricbeat`
- `{internal-name}` is the logical name (`spec.name` or the resource name)

For example with `name: production-kafka`, certificates are mounted at `/usr/share/filebeat/discover/production-kafka/` for Filebeat and `/usr/share/logstash/discover/production-kafka/` for Logstash.

## Status Information

```yaml
status:
  phase: Ready
  conditions:
    - type: Ready
      status: "True"
  secretFileRef: kafka-sample-file
  secretEnvRef: kafka-sample-env
  kafkaCASecretRef: my-kafka-cluster-cluster-ca-cert
  kafkaUserSecretRef: my-kafka-user
```

## Usage Examples

### Managed Kafka with TLS authentication

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: production-kafka
  namespace: cluster-dev
spec:
  kafkaRef:
    managed:
      name: prod-kafka-cluster
      namespace: kafka-system
      targetListener: tls
      userRef:
        name: app-user
```

### Kafka with a custom environment variable suffix

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: my-app-kafka
spec:
  name: MYAPP_PROD
  kafkaRef:
    managed:
      name: prod-kafka-cluster
      userRef:
        name: myapp-user
```

This generates variables such as `KAFKA_BOOTSTRAP_SERVERS_MYAPP_PROD`, `KAFKA_AUTH_TYPE_MYAPP_PROD` and `KAFKA_CA_TRUSTSTORE_PASSWORD_MYAPP_PROD`.

### External Kafka with client certificates

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: external-kafka
spec:
  kafkaRef:
    external:
      addresses:
        - kafka1.external.com:9093
        - kafka2.external.com:9093
      userSecretRef:
        name: kafka-client-certs
    kafkaCASecretRef:
      name: external-kafka-ca
```

### Simple external Kafka

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: dev-kafka
spec:
  kafkaRef:
    external:
      addresses:
        - localhost:9092
```

## Integration with Filebeat and Logstash

The Kafka Discover resource is referenced from Filebeat and Logstash through `discoverRef`:

```yaml
spec:
  discoverRef:
    - kafka:
        name: production-kafka
```

See [Filebeat Discover settings](../filebeat/discover-settings.md) and [Logstash Discover settings](../logstash/discover-settings.md) for integration details.

## Security Considerations

- Generated secrets contain sensitive information (private keys, keystore passwords). Restrict access with RBAC.
- Keystore and truststore passwords are randomly generated and stored in the env secret.
- The operator preserves generated keystore/truststore data across reconciliations (it does not regenerate them on every loop), so consumers must rely on the generated passwords to open them.
- Update the referenced CA/user secrets to rotate certificates; the operator propagates changes to the generated secrets and restarts dependent workloads.
