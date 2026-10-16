# Logstash Discover settings

Logstash can consume the Discover concept to receive connection information (certificates, keystores, environment variables) from Discover resources. The operator does not generate the Logstash pipelines: you reference the discovered information in your own pipeline configuration.

## Overview

The Discover integration in Logstash provides:
- Centralized connection and certificate management for Kafka, Logstash and Elasticsearch
- SSL/TLS certificate management for secure connections
- Environment variables and file-based certificates injected into the pods
- Seamless integration with Logstash pipelines

For detailed information about each Discover resource, see:
- [Kafka Discover](../discover/kafka.md)
- [Logstash Discover](../discover/logstash.md)
- [Elasticsearch Discover](../discover/elasticsearch.md)

> The usage of `elasticsearchRef` is deprecated in favor of `discoverRef`. Both patterns cannot be used at the same time.

## Configuration

Add the Discover references in your Logstash spec:

- **discoverRef** (slice of object): References to Discover resources in the same namespace. Each entry references exactly one Discover resource:
  - **kafka** (object): `name` (string) of a Kafka Discover resource
  - **logstash** (object): `name` (string) of a Logstash Discover resource
  - **elasticsearch** (object): `name` (string) of an Elasticsearch Discover resource

All referenced Discover resources are injected into the Logstash pods: the env secret is injected with `envFrom` and the file secret is mounted under `/usr/share/logstash/discover/{name}/`.

The operator restarts Logstash pods when a referenced Discover secret changes.

### Environment variables

The env secret of each referenced Discover resource is injected into the Logstash container with `envFrom`. Variable names use the service prefix and a suffix derived from the Discover logical name (`spec.name` or resource name, uppercased with `-` replaced by `_`):

- Kafka: `KAFKA_BOOTSTRAP_SERVERS_{SUFFIX}`, `KAFKA_AUTH_TYPE_{SUFFIX}`, `KAFKA_OAUTH_*_{SUFFIX}`, `KAFKA_CA_TRUSTSTORE_PASSWORD_{SUFFIX}`, `KAFKA_USER_PASSWORD_{SUFFIX}`
- Logstash: `LOGSTASH_HOSTS_{SUFFIX}`, `LOGSTASH_USER_PASSWORD_{SUFFIX}`
- Elasticsearch: `ELASTICSEARCH_HOSTS_{SUFFIX}`, `ELASTICSEARCH_USERNAME_{SUFFIX}`, `ELASTICSEARCH_PASSWORD_{SUFFIX}`

Example: a Discover resource named `production-kafka` exposes `KAFKA_BOOTSTRAP_SERVERS_PRODUCTION_KAFKA`, usable in a pipeline as `${KAFKA_BOOTSTRAP_SERVERS_PRODUCTION_KAFKA}`.

### Mounted files

The file secret of each referenced Discover resource is mounted at:

`/usr/share/logstash/discover/{discover-name}/`

where `{discover-name}` is the Discover logical name. Use these paths in your pipelines, for example:

- Kafka: `ca.crt`, `ca.p12` (truststore), `user.crt`, `user.key`, `user.p12` (keystore)
- Logstash: `ca.crt`, `user.crt`, `user.key`, `user.p12`
- Elasticsearch: `ca.crt`

The generated PKCS#12 passwords are exposed through `KAFKA_CA_TRUSTSTORE_PASSWORD_{SUFFIX}`, `KAFKA_USER_PASSWORD_{SUFFIX}` and `LOGSTASH_USER_PASSWORD_{SUFFIX}`.

## Complete examples

### Logstash consuming a Kafka input

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
        name: logstash-user
---
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  version: 8.6.0
  discoverRef:
    - kafka:
        name: production-kafka
  pipelines:
    kafka-input.conf: |
      input {
        kafka {
          bootstrap_servers => "${KAFKA_BOOTSTRAP_SERVERS_PRODUCTION_KAFKA}"
          topics => ["application-logs"]
          group_id => "logstash-production"
          security_protocol => "SSL"
          ssl_truststore_location => "/usr/share/logstash/discover/production-kafka/ca.p12"
          ssl_truststore_password => "${KAFKA_CA_TRUSTSTORE_PASSWORD_PRODUCTION_KAFKA}"
          ssl_truststore_type => "PKCS12"
          ssl_keystore_location => "/usr/share/logstash/discover/production-kafka/user.p12"
          ssl_keystore_password => "${KAFKA_USER_PASSWORD_PRODUCTION_KAFKA}"
          ssl_keystore_type => "PKCS12"
          codec => json
        }
      }
      output {
        stdout { codec => rubydebug }
      }
```

### Logstash sending to Elasticsearch

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
    secretRef:
      name: elasticsearch-credentials
---
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  version: 8.6.0
  discoverRef:
    - elasticsearch:
        name: production-elasticsearch
  pipelines:
    output.conf: |
      input {
        beats {
          port => 5044
        }
      }
      output {
        elasticsearch {
          hosts => "${ELASTICSEARCH_HOSTS_PRODUCTION_ELASTICSEARCH}"
          index => "logs-%{+YYYY.MM.dd}"
          user => "${ELASTICSEARCH_USERNAME_PRODUCTION_ELASTICSEARCH}"
          password => "${ELASTICSEARCH_PASSWORD_PRODUCTION_ELASTICSEARCH}"
          ssl_enabled => true
          ssl_certificate_authorities => ["/usr/share/logstash/discover/production-elasticsearch/ca.crt"]
        }
      }
```

## Migration from elasticsearchRef

**Before (deprecated static ref):**

```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
    secretRef:
      name: elasticsearch-credentials
```

The legacy pattern exposes `ELASTICSEARCH_HOST`, `ELASTICSEARCH_CA_PATH`, `ELASTICSEARCH_USERNAME` and `ELASTICSEARCH_PASSWORD`.

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
# Step 2: Reference it from Logstash
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
spec:
  discoverRef:
    - elasticsearch:
        name: my-elasticsearch-discover
  pipelines:
    output.conf: |
      input {
        beats {
          port => 5044
        }
      }
      output {
        elasticsearch {
          hosts => "${ELASTICSEARCH_HOSTS_MY_ELASTICSEARCH_DISCOVER}"
          index => "logs-%{+YYYY.MM.dd}"
          user => "${ELASTICSEARCH_USERNAME_MY_ELASTICSEARCH_DISCOVER}"
          password => "${ELASTICSEARCH_PASSWORD_MY_ELASTICSEARCH_DISCOVER}"
          ssl_enabled => true
          ssl_certificate_authorities => ["/usr/share/logstash/discover/my-elasticsearch-discover/ca.crt"]
        }
      }
```
