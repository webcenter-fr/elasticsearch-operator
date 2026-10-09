# Filebeat Discover settings

Filebeat can consume the Discover concept to automatically configure its output and to receive connection information (certificates, keystores, environment variables) from Discover resources. The supported outputs are Kafka and Logstash.

## Overview

The Discover integration in Filebeat provides:
- Automatic Kafka or Logstash output configuration based on the referenced Discover resource
- SSL/TLS certificate management for secure connections
- Support for both managed (Strimzi / this operator) and external clusters
- Environment variables and file-based certificates injected into the pods

For detailed information about each Discover resource, see:
- [Kafka Discover](../discover/kafka.md)
- [Logstash Discover](../discover/logstash.md)
- [Elasticsearch Discover](../discover/elasticsearch.md)

> The usage of `elasticsearchRef` and `logstashRef` is deprecated in favor of `discoverRef`. Both patterns cannot be used at the same time.

## Configuration

Add the Discover references in your Filebeat spec:

- **discoverRef** (slice of object): References to Discover resources in the same namespace. Each entry references exactly one Discover resource:
  - **kafka** (object): `name` (string) of a Kafka Discover resource
  - **logstash** (object): `name` (string) of a Logstash Discover resource
  - **elasticsearch** (object): `name` (string) of an Elasticsearch Discover resource
- **discoverOutputName** (string): The name of the Discover reference (one of `discoverRef`) used to auto-generate the Filebeat output. It must match the `name` of one `discoverRef` entry. Only `kafka` and `logstash` Discover resources generate an output.

All referenced Discover resources are injected into the Filebeat pods, even those not used as output: the env secret is injected with `envFrom` and the file secret is mounted under `/usr/share/filebeat/discover/{name}/`.

The operator restarts Filebeat pods when a referenced Discover secret changes.

### Auto-generated outputs

When `discoverOutputName` references a Kafka Discover resource, the operator generates the following `output.kafka` section (fields already set in `config` / `extraConfigs` are kept):

```yaml
output.kafka:
  enabled: true
  client_id: ${POD_NAME}
  compression: lz4
  hosts:
    - ${KAFKA_BOOTSTRAP_SERVERS_<SUFFIX>}
  partition.round_robin:
    reachable_only: true
  required_acks: 1
  ssl:
    enabled: true
    certificate_authorities:
      - /usr/share/filebeat/discover/{discover-name}/ca.crt
    certificate: /usr/share/filebeat/discover/{discover-name}/user.crt
    key: /usr/share/filebeat/discover/{discover-name}/user.key
    verification_mode: full
```

The SSL client certificate/key entries are only added when the file secret contains `user.crt` and `user.key`. The `ssl` block stays disabled when no CA is available.

When `discoverOutputName` references a Logstash Discover resource, the operator generates the following `output.logstash` section:

```yaml
output.logstash:
  enabled: true
  hosts:
    - ${LOGSTASH_HOSTS_<SUFFIX>}
  loadbalance: true
  ssl:
    enabled: true
    certificate_authorities:
      - /usr/share/filebeat/discover/{discover-name}/ca.crt
    certificate: /usr/share/filebeat/discover/{discover-name}/user.crt
    key: /usr/share/filebeat/discover/{discover-name}/user.key
    verification_mode: full
```

### Environment variables

The env secret of each referenced Discover resource is injected into the Filebeat container with `envFrom`. Variable names use the service prefix and a suffix derived from the Discover logical name (`spec.name` or resource name, uppercased with `-` replaced by `_`):

- Kafka: `KAFKA_BOOTSTRAP_SERVERS_{SUFFIX}`, `KAFKA_AUTH_TYPE_{SUFFIX}`, `KAFKA_OAUTH_*_{SUFFIX}`, `KAFKA_CA_TRUSTSTORE_PASSWORD_{SUFFIX}`, `KAFKA_USER_PASSWORD_{SUFFIX}`
- Logstash: `LOGSTASH_HOSTS_{SUFFIX}`, `LOGSTASH_USER_PASSWORD_{SUFFIX}`
- Elasticsearch: `ELASTICSEARCH_HOSTS_{SUFFIX}`, `ELASTICSEARCH_USERNAME_{SUFFIX}`, `ELASTICSEARCH_PASSWORD_{SUFFIX}`

Example: a Discover resource named `production-kafka` exposes `KAFKA_BOOTSTRAP_SERVERS_PRODUCTION_KAFKA`.

### Mounted files

The file secret of each referenced Discover resource is mounted at:

`/usr/share/filebeat/discover/{discover-name}/`

where `{discover-name}` is the Discover logical name. Use these paths in your Filebeat SSL settings, for example `/usr/share/filebeat/discover/production-kafka/ca.crt`.

Common files:

- Kafka: `ca.crt`, `ca.p12`, `user.crt`, `user.key`, `user.p12`
- Logstash: `ca.crt`, `user.crt`, `user.key`, `user.p12`
- Elasticsearch: `ca.crt`

## Complete examples

### Filebeat with a managed Kafka output

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Kafka
metadata:
  name: production-kafka
  namespace: cluster-dev
spec:
  kafkaRef:
    managed:
      name: kafka-cluster
      targetListener: tls
      userRef:
        name: filebeat-user
---
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
  namespace: cluster-dev
spec:
  version: 8.6.0
  discoverRef:
    - kafka:
        name: production-kafka
  discoverOutputName: production-kafka
  config:
    filebeat.inputs:
      - type: log
        paths:
          - /var/log/app/*.log
        fields:
          service: myapp
          environment: production
        fields_under_root: true
  deployment:
    replicas: 1
```

### Filebeat with an external Logstash output

```yaml
apiVersion: discover.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: external-logstash
  namespace: cluster-dev
spec:
  logstashRef:
    external:
      addresses:
        - logstash1.example.com:5044
        - logstash2.example.com:5044
    logstashCASecretRef:
      name: external-logstash-ca
---
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
  namespace: cluster-dev
spec:
  version: 8.6.0
  discoverRef:
    - logstash:
        name: external-logstash
  discoverOutputName: external-logstash
  config:
    filebeat.inputs:
      - type: log
        paths:
          - /var/log/*.log
```

### Filebeat referencing several Discover resources

All references are mounted and injected; only the one matching `discoverOutputName` generates the output. This is useful to consume extra connection information (for example an Elasticsearch Discover) in inputs or processors:

```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
spec:
  version: 8.6.0
  discoverRef:
    - logstash:
        name: production-logstash
    - elasticsearch:
        name: production-elasticsearch
  discoverOutputName: production-logstash
  config:
    filebeat.inputs:
      - type: log
        paths:
          - /var/log/*.log
```

## Migration from elasticsearchRef / logstashRef

**Before (deprecated static ref):**

```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
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
---
# Step 2: Reference it from Filebeat
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
spec:
  discoverRef:
    - elasticsearch:
        name: my-elasticsearch-discover
  discoverOutputName: my-elasticsearch-discover
```
