# Elasticsearch Discover

The Elasticsearch Discover resource provides a unified way to manage Elasticsearch connection information for workloads (Logstash, Metricbeat). It generates and maintains secrets containing the Elasticsearch CA certificate, URL and credentials.

## Overview

The Elasticsearch Discover resource automatically:
- Generates connection secrets with file-based (CA certificate) and environment variable (hosts, username, password) formats
- Collects the CA certificate of a managed Elasticsearch cluster (when its TLS is self-managed) and any custom CA
- Supports both Elasticsearch clusters managed by this operator and external Elasticsearch clusters
- Watches for changes and updates dependent resources automatically (workloads are restarted when the secrets change)

## Resource Definition

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch-sample
  namespace: cluster-dev
spec:
  # Target secrets configuration
  targetSecretRef:
    name: elasticsearch-sample-file        # Optional: defaults to {name}-file
  targetSecretEnvRef:
    name: elasticsearch-sample-env         # Optional: defaults to {name}-env
  name: ELASTICSEARCH_SAMPLE               # Optional: suffix for env vars / mount dir, defaults to resource name

  # Elasticsearch connection reference
  elasticsearchRef:
    # For managed Elasticsearch (this operator)
    managed:
      name: my-elasticsearch
      namespace: cluster-dev       # Optional: defaults to same namespace
      targetNodeGroup: client      # Optional: use a specific node group service

    # OR for external Elasticsearch
    external:
      addresses:
        - https://elasticsearch.example.com:9200

    # Optional: credentials secret (username / password)
    secretRef:
      name: elasticsearch-credentials

    # Optional: custom CA certificate (ca.crt)
    elasticsearchCASecretRef:
      name: custom-elasticsearch-ca
```

## Configuration Options

### Target Secrets

| Field | Description | Default |
|-------|-------------|---------|
| `targetSecretRef.name` | Secret name for file-based information (CA certificate) | `{resource-name}-file` |
| `targetSecretEnvRef.name` | Secret name for environment variables | `{resource-name}-env` |
| `name` | Logical name used as suffix for environment variables and as directory name to mount the file secret on pods | Resource name |

> `name` must start with an alphanumeric character and only contain alphanumeric characters, `-`, `_` or `.`. It must not contain `..`.

### Elasticsearch Reference

| Field | Description | Required |
|-------|-------------|----------|
| `elasticsearchRef.managed.name` | Name of the `Elasticsearch` resource | One of managed/external |
| `elasticsearchRef.managed.namespace` | Namespace of the `Elasticsearch` resource | No (defaults to same namespace) |
| `elasticsearchRef.managed.targetNodeGroup` | Node group service to use. Default: the global service | No |
| `elasticsearchRef.external.addresses` | List of Elasticsearch URLs (http/https) | One of managed/external |
| `elasticsearchRef.secretRef.name` | Secret containing `username` and `password` keys | No |
| `elasticsearchRef.elasticsearchCASecretRef.name` | Secret containing a custom CA certificate (`ca.crt`) | No |

For a managed Elasticsearch cluster, the operator uses the public URL of the cluster (or of the target node group) and, when the TLS is enabled and self-managed, automatically collects its CA certificate.

## Generated Secrets

The Elasticsearch Discover resource generates two secrets, owned by the `Elasticsearch` resource.

### File Secret (`{name}-file`)

| Key | Description |
|-----|-------------|
| `ca.crt` | Concatenation of the managed Elasticsearch CA and custom CA certificates (the key always exists, it may be empty when no CA is configured) |

### Environment Variable Secret (`{name}-env`)

Contains connection information as environment variables. The suffix (`{SUFFIX}`) is the logical name (`spec.name` or the resource name) uppercased with `-` replaced by `_`:

| Variable | Description |
|----------|-------------|
| `ELASTICSEARCH_HOSTS_{SUFFIX}` | Comma-separated list of Elasticsearch URLs |
| `ELASTICSEARCH_USERNAME_{SUFFIX}` | Username, when `secretRef` is configured |
| `ELASTICSEARCH_PASSWORD_{SUFFIX}` | Password, when `secretRef` is configured |

For example, with a resource named `elasticsearch-sample` (and no `spec.name`), the hosts variable is `ELASTICSEARCH_HOSTS_ELASTICSEARCH_SAMPLE`.

### Secret Metadata

Both secrets carry the following labels and annotations, used by workloads to select and mount them:

```yaml
labels:
  discoverName: elasticsearch-sample
  discover.k8s.webcenter.fr: "true"
  elasticsearch.discover.k8s.webcenter.fr: "true"
annotations:
  discover.k8s.webcenter.fr: "true"
  discover.k8s.webcenter.fr/type: "env"   # or "file"
  discover.k8s.webcenter.fr/mountPath: elasticsearch-sample
  elasticsearch.discover.k8s.webcenter.fr: "true"
```

## Certificate File Mounting

The file secret of each referenced Discover resource is mounted into the workload pods at:

`/usr/share/{application}/discover/{internal-name}/`

where:
- `{application}` is `filebeat`, `logstash` or `metricbeat`
- `{internal-name}` is the logical name (`spec.name` or the resource name)

For example with `name: production-elasticsearch`, the CA certificate is mounted at `/usr/share/metricbeat/discover/production-elasticsearch/ca.crt`.

## Status Information

```yaml
status:
  phase: Ready
  conditions:
    - type: Ready
      status: "True"
  secretFileRef: elasticsearch-sample-file
  secretEnvRef: elasticsearch-sample-env
  elasticsearchCASecretRef: my-elasticsearch-tls-api
  elasticsearchUserSecretRef: elasticsearch-credentials
```

## Usage Examples

### Managed Elasticsearch

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: production-elasticsearch
  namespace: cluster-dev
spec:
  elasticsearchRef:
    managed:
      name: elasticsearch
      targetNodeGroup: client
```

### External Elasticsearch with credentials

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: external-elasticsearch
spec:
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch.example.com:9200
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: external-elasticsearch-ca
```

### Elasticsearch with a custom environment variable suffix

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: my-app-elasticsearch
spec:
  name: MYAPP_PROD
  elasticsearchRef:
    managed:
      name: elasticsearch
```

This generates variables such as `ELASTICSEARCH_HOSTS_MYAPP_PROD`, `ELASTICSEARCH_USERNAME_MYAPP_PROD` and `ELASTICSEARCH_PASSWORD_MYAPP_PROD`.

## Integration with Workloads

The Elasticsearch Discover resource is referenced from Logstash and Metricbeat through `discoverRef`:

```yaml
spec:
  discoverRef:
    - elasticsearch:
        name: production-elasticsearch
```

Metricbeat can use it as output (`discoverOutputName`), while Logstash can consume the exposed environment variables and mounted CA certificate in its pipelines. See [Logstash Discover settings](../logstash/discover-settings.md) and [Metricbeat Discover settings](../metricbeat/discover-settings.md).

## Security Considerations

- The credentials secret is copied by reference into the env secret; it contains the Elasticsearch username and password. Restrict access with RBAC.
- Update the referenced CA/credentials secrets to rotate them; the operator propagates changes to the generated secrets and restarts dependent workloads.
