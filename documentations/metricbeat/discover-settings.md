# Metricbeat Discover settings

Metricbeat can consume the Discover concept to automatically configure its Elasticsearch output and to receive connection information (CA certificate, hosts, credentials) from a Discover resource.

## Overview

The Discover integration in Metricbeat provides:
- Automatic Elasticsearch output configuration based on the referenced Elasticsearch Discover resource
- SSL/TLS certificate management for secure connections
- Environment variables and file-based certificates injected into the pods

For detailed information about each Discover resource, see:
- [Kafka Discover](../discover/kafka.md)
- [Logstash Discover](../discover/logstash.md)
- [Elasticsearch Discover](../discover/elasticsearch.md)

> The usage of `elasticsearchRef` is deprecated in favor of `discoverRef` and `discoverOutputName`. Both patterns cannot be used at the same time.

## Configuration

Add the Discover references in your Metricbeat spec:

- **discoverRef** (slice of object): References to Discover resources in the same namespace. Each entry references exactly one Discover resource:
  - **kafka** (object): `name` (string) of a Kafka Discover resource
  - **logstash** (object): `name` (string) of a Logstash Discover resource
  - **elasticsearch** (object): `name` (string) of an Elasticsearch Discover resource
- **discoverOutputName** (string): The name of the Discover reference (one of `discoverRef`) used to auto-generate the Metricbeat output. It must match the `name` of one `discoverRef` entry and reference an Elasticsearch Discover resource.

All referenced Discover resources are injected into the Metricbeat pods: the env secret is injected with `envFrom` and the file secret is mounted under `/usr/share/metricbeat/discover/{name}/`.

The operator restarts Metricbeat pods when a referenced Discover secret changes.

### Auto-generated output

When `discoverOutputName` references an Elasticsearch Discover resource, the operator generates the following `output.elasticsearch` section (fields already set in `config` / `extraConfigs` are kept):

```yaml
output.elasticsearch:
  hosts:
    - ${ELASTICSEARCH_HOSTS_<SUFFIX>}
  username: ${ELASTICSEARCH_USERNAME_<SUFFIX>}
  password: ${ELASTICSEARCH_PASSWORD_<SUFFIX>}
  ssl:
    enabled: true
    certificate_authorities:
      - /usr/share/metricbeat/discover/{discover-name}/ca.crt
```

Username and password entries are only added when the env secret exposes the corresponding credentials. The `ssl` block is only added when the file secret contains `ca.crt`.

### Environment variables

The env secret of each referenced Discover resource is injected into the Metricbeat container with `envFrom`. Variable names use the service prefix and a suffix derived from the Discover logical name (`spec.name` or resource name, uppercased with `-` replaced by `_`):

- Elasticsearch: `ELASTICSEARCH_HOSTS_{SUFFIX}`, `ELASTICSEARCH_USERNAME_{SUFFIX}`, `ELASTICSEARCH_PASSWORD_{SUFFIX}`
- Kafka: `KAFKA_BOOTSTRAP_SERVERS_{SUFFIX}`, `KAFKA_AUTH_TYPE_{SUFFIX}`, ...
- Logstash: `LOGSTASH_HOSTS_{SUFFIX}`, ...

Example: a Discover resource named `production-elasticsearch` exposes `ELASTICSEARCH_HOSTS_PRODUCTION_ELASTICSEARCH`.

### Mounted files

The file secret of each referenced Discover resource is mounted at:

`/usr/share/metricbeat/discover/{discover-name}/`

where `{discover-name}` is the Discover logical name. An Elasticsearch Discover resource exposes `ca.crt`; use it in your SSL settings, for example `/usr/share/metricbeat/discover/production-elasticsearch/ca.crt`.

## Complete example

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
    secretRef:
      name: elasticsearch-credentials
---
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  name: metricbeat
  namespace: cluster-dev
spec:
  version: 8.6.0
  discoverRef:
    - elasticsearch:
        name: production-elasticsearch
  discoverOutputName: production-elasticsearch
  config:
    tags: ["service-X", "web-tier"]
  modules:
    system.yml:
      - module: system
        metricsets: ["cpu", "memory", "network"]
        period: 10s
        enabled: true
```

## Migration from elasticsearchRef

**Before (deprecated static ref):**

```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  name: metricbeat
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
    secretRef:
      name: elasticsearch-credentials
```

**After (Discover):**

```yaml
# Step 1: Create the Discover resource
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: my-elasticsearch-discover
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
    secretRef:
      name: elasticsearch-credentials
---
# Step 2: Reference it from Metricbeat
apiVersion: beat.k8s.webcenter.fr/v1
kind: Metricbeat
metadata:
  name: metricbeat
spec:
  discoverRef:
    - elasticsearch:
        name: my-elasticsearch-discover
  discoverOutputName: my-elasticsearch-discover
```
