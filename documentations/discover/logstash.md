# Logstash Discover

The Logstash Discover resource provides a unified way to manage Logstash connection information for workloads that send data to Logstash (typically Filebeat). It generates and maintains secrets containing CA certificates, client certificates/keystores and the list of Logstash hosts.

## Overview

The Logstash Discover resource automatically:
- Generates connection secrets with file-based (certificates, keystores) and environment variable (hosts, generated passwords) formats
- Manages SSL/TLS certificates and PKCS#12 keystores for secure connections to the Logstash beat input
- Supports both Logstash clusters managed by this operator and external Logstash clusters
- Watches for changes and updates dependent resources automatically (workloads are restarted when the secrets change)

## Resource Definition

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash-sample
  namespace: cluster-dev
spec:
  # Target secrets configuration
  targetSecretRef:
    name: logstash-sample-file        # Optional: defaults to {name}-file
  targetSecretEnvRef:
    name: logstash-sample-env         # Optional: defaults to {name}-env
  name: LOGSTASH_SAMPLE               # Optional: suffix for env vars / mount dir, defaults to resource name

  # Logstash connection reference
  logstashRef:
    # For managed Logstash (this operator)
    managed:
      name: my-logstash
      namespace: log-management       # Optional: defaults to same namespace
      targetService: beat             # Optional: target service that exposes the beat protocol
      port: 5003                      # Required: port of the beat input
      userRef:
        name: logstash-user-secret    # Optional: secret that stores user certificates

    # OR for external Logstash
    external:
      addresses:
        - logstash1.example.com:5044
        - logstash2.example.com:5044
      userSecretRef:
        name: logstash-client-certs   # Optional: SSL client authentication

    # Optional: Custom CA certificates
    logstashCASecretRef:
      name: custom-logstash-ca
```

## Configuration Options

### Target Secrets

| Field | Description | Default |
|-------|-------------|---------|
| `targetSecretRef.name` | Secret name for file-based information (certificates, private keys, keystores) | `{resource-name}-file` |
| `targetSecretEnvRef.name` | Secret name for environment variables | `{resource-name}-env` |
| `name` | Logical name used as suffix for environment variables and as directory name to mount the file secret on pods | Resource name |

> `name` must start with an alphanumeric character and only contain alphanumeric characters, `-`, `_` or `.`. It must not contain `..`.

### Managed Logstash Reference

For Logstash clusters managed by this operator:

| Field | Description | Required |
|-------|-------------|----------|
| `managed.name` | Name of the `Logstash` resource | Yes |
| `managed.namespace` | Namespace of the `Logstash` resource | No (defaults to same namespace) |
| `managed.targetService` | Name of the service that exposes the beat protocol. When set, its URL is used instead of the per-replica URLs | No |
| `managed.port` | Port of the beat input | Yes |
| `managed.userRef` | Secret reference (`name`, optionally `namespace`) that stores user certificates (`user.crt`, `user.key`, `ca.crt`) | No |

When `targetService` is not set, the operator computes one host per Logstash replica with the form:

`{statefulset-name}-{index}.{global-service-name}.{namespace}.svc:{port}`

When the managed Logstash has PKI enabled, its CA certificate is collected automatically into the file secret.

### External Logstash Reference

For Logstash clusters not managed by this operator:

| Field | Description | Required |
|-------|-------------|----------|
| `external.addresses` | List of Logstash addresses (`host:port`) | Yes |
| `external.userSecretRef.name` | Secret containing client SSL certificate and key | No |

The user secret may contain:
- `user.crt`: Client certificate
- `user.key`: Client private key
- `ca.crt`: CA certificate used to sign the client certificate (optional)

### CA Certificate Reference

| Field | Description |
|-------|-------------|
| `logstashCASecretRef.name` | Secret containing custom CA certificates. Every entry ending by `.crt` or `.pem` is added to the generated CA bundle |

## Generated Secrets

The Logstash Discover resource generates two secrets, owned by the `Logstash` resource.

### File Secret (`{name}-file`)

Contains certificates and keystores as files:

| Key | Description |
|-----|-------------|
| `ca.crt` | Concatenation of the managed Logstash PKI CA and custom CA certificates (the key always exists, it may be empty when no CA is configured) |
| `user.crt` | User certificate, when a user secret is configured |
| `user.key` | User private key, when a user secret is configured |
| `user.p12` | PKCS#12 keystore generated from the user certificate and private key |

### Environment Variable Secret (`{name}-env`)

Contains connection information as environment variables. The suffix (`{SUFFIX}`) is the logical name (`spec.name` or the resource name) uppercased with `-` replaced by `_`:

| Variable | Description |
|----------|-------------|
| `LOGSTASH_HOSTS_{SUFFIX}` | Comma-separated list of `host:port` to reach the beat input |
| `LOGSTASH_USER_PASSWORD_{SUFFIX}` | Generated password of the user keystore (`user.p12`), when a user certificate is discovered |

For example, with a resource named `logstash-sample` (and no `spec.name`), the hosts variable is `LOGSTASH_HOSTS_LOGSTASH_SAMPLE`.

### Secret Metadata

Both secrets carry the following labels and annotations, used by workloads to select and mount them:

```yaml
labels:
  discoverName: logstash-sample
  discover.k8s.webcenter.fr: "true"
  logstash.discover.k8s.webcenter.fr: "true"
annotations:
  discover.k8s.webcenter.fr: "true"
  discover.k8s.webcenter.fr/type: "env"   # or "file"
  discover.k8s.webcenter.fr/mountPath: logstash-sample
  logstash.discover.k8s.webcenter.fr: "true"
```

## Certificate File Mounting

The file secret of each referenced Discover resource is mounted into the workload pods at:

`/usr/share/{application}/discover/{internal-name}/`

where:
- `{application}` is `filebeat`, `logstash` or `metricbeat`
- `{internal-name}` is the logical name (`spec.name` or the resource name)

For example with `name: production-logstash`, certificates are mounted at `/usr/share/filebeat/discover/production-logstash/`.

## Status Information

```yaml
status:
  phase: Ready
  conditions:
    - type: Ready
      status: "True"
  secretFileRef: logstash-sample-file
  secretEnvRef: logstash-sample-env
  logstashCASecretRef: my-logstash-tls
  logstashUserSecretRef: logstash-user-secret
```

## Usage Examples

### Managed Logstash

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: production-logstash
  namespace: cluster-dev
spec:
  logstashRef:
    managed:
      name: logstash-log
      targetService: beat
      port: 5003
```

### External Logstash with client certificates

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: external-logstash
spec:
  logstashRef:
    external:
      addresses:
        - logstash1.external.com:5044
        - logstash2.external.com:5044
      userSecretRef:
        name: logstash-client-certs
    logstashCASecretRef:
      name: external-logstash-ca
```

### Logstash with a custom environment variable suffix

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: my-app-logstash
spec:
  name: MYAPP_PROD
  logstashRef:
    managed:
      name: logstash-log
      port: 5003
```

This generates the variable `LOGSTASH_HOSTS_MYAPP_PROD`.

## Integration with Filebeat and Logstash

The Logstash Discover resource is referenced from Filebeat and Logstash through `discoverRef`:

```yaml
spec:
  discoverRef:
    - logstash:
        name: production-logstash
```

Filebeat can use it as output (`discoverOutputName`), while Logstash can consume the exposed environment variables and mounted certificates in its pipelines. See [Filebeat Discover settings](../filebeat/discover-settings.md) and [Logstash Discover settings](../logstash/discover-settings.md).

## Security Considerations

- Generated secrets contain sensitive information (private keys, keystore passwords). Restrict access with RBAC.
- Keystore passwords are randomly generated and stored in the env secret.
- Update the referenced CA/user secrets to rotate certificates; the operator propagates changes to the generated secrets and restarts dependent workloads.
