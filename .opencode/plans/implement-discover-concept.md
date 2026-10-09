# Plan: Implement Discover Concept in Elasticsearch Operator

## Overview

Implement a **Discover** pattern in the elasticsearch-operator to provide automatic discovery and distribution of connection information (certificates, credentials, URLs, keystores, truststores) for external services (Kafka, Logstash, Elasticsearch) to workloads (Filebeat, Logstash, Metricbeat).

This is a port of the existing Discover implementation from `opensearch-operator-k8s` to `elasticsearch-operator`.

## Current State vs Target State

### Current State (elasticsearch-operator)
- **Static references**: Workloads use `elasticsearchRef` and `logstashRef` directly in their specs
- **No Kafka support**: No Strimzi integration, no Kafka discovery
- **Manual certificate management**: Users must manually provide CA secrets and credentials
- **Hardcoded config generation**: Operator generates `filebeat.yml` with hardcoded output config

### Target State (with Discover)
- **Dynamic discovery**: Discover CRDs (Kafka, Logstash, Elasticsearch) generate connection secrets
- **Kafka support**: Full Strimzi integration for managed Kafka clusters
- **Automatic secret management**: Discover controllers generate and maintain all secrets (certs, keystores, passwords)
- **Flexible workload integration**: Workloads reference Discover CRs via `discoverRef`
- **Backward compatible**: Existing `elasticsearchRef`/`logstashRef` continue to work (deprecated)

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DISCOVER CRDs                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────────┐  │
│  │ Kafka       │  │ Logstash    │  │ Elasticsearch                   │  │
│  │ (discover)  │  │ (discover)  │  │ (discover)                      │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────────────────┘  │
│         │                │                    │                         │
│  ┌──────▼────────────────▼────────────────────▼──────────────────────┐  │
│  │              Discover Controllers (3 controllers)                 │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────────┐ │  │
│  │  │ Kafka       │ │ Logstash    │ │ Elasticsearch               │ │  │
│  │  │ Controller  │ │ Controller  │ │ Controller                  │ │  │
│  │  └──────┬──────┘ └──────┬──────┘ └──────────┬──────────────────┘ │  │
│  └─────────┼───────────────┼────────────────────┼────────────────────┘  │
│            │               │                    │                       │
│  ┌─────────▼───────────────▼────────────────────▼────────────────────┐  │
│  │              Generated Secrets (2 per Discover CR)                │  │
│  │  ┌─────────────────┐    ┌─────────────────┐                      │  │
│  │  │ {name}-env      │    │ {name}-file     │                      │  │
│  │  │ (env variables) │    │ (certificates)  │                      │  │
│  │  │                 │    │                 │                      │  │
│  │  │ KAFKA_BOOTSTRAP │    │ ca.crt          │                      │  │
│  │  │ KAFKA_AUTH_TYPE │    │ ca.p12          │                      │  │
│  │  │ KAFKA_USER_PWD  │    │ user.crt        │                      │  │
│  │  │ KAFKA_CA_PWD    │    │ user.key        │                      │  │
│  │  │ LOGSTASH_HOSTS  │    │ user.p12        │                      │  │
│  │  │ ELASTICSEARCH_  │    │                 │                      │  │
│  │  │   HOSTS         │    │                 │                      │  │
│  │  │ ELASTICSEARCH_  │    │                 │                      │  │
│  │  │   USERNAME      │    │                 │                      │  │
│  │  │ ELASTICSEARCH_  │    │                 │                      │  │
│  │  │   PASSWORD      │    │                 │                      │  │
│  │  └─────────────────┘    └─────────────────┘                      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                              │                                          │
│  ┌───────────────────────────▼───────────────────────────────────────┐  │
│  │              Workloads (Filebeat / Logstash / Metricbeat CRs)     │  │
│  │                                                                   │  │
│  │  spec.discoverRef:                                                │  │
│  │    - kafka: { name: my-kafka }                                    │  │
│  │    - logstash: { name: my-logstash }                              │  │
│  │    - elasticsearch: { name: my-elasticsearch }                    │  │
│  │                                                                   │  │
│  │  ┌─────────────────────────────────────────────────────────────┐ │  │
│  │  │  StatefulSet Pod Template                                    │ │  │
│  │  │  ┌─────────────┐  ┌─────────────────────────────────────┐  │ │  │
│  │  │  │ envFrom:    │  │ volumes:                            │  │ │  │
│  │  │  │ - secretRef:│  │ - name: kafka-test-file             │  │ │  │
│  │  │  │   {name}-env│  │   secret: {name}-file               │  │ │  │
│  │  │  │             │  │                                     │  │ │  │
│  │  │  │ volumeMounts│  │ volumeMounts:                       │  │ │  │
│  │  │  │ - name: ... │  │ - name: kafka-test-file             │  │ │  │
│  │  │  │   mountPath:│  │   mountPath: /usr/share/{app}/      │  │ │  │
│  │  │  │   /usr/share/│  │              discover/{name}/       │  │ │  │
│  │  │  │   {app}/... │  │                                     │  │ │  │
│  │  └─────────────┘  └─────────────────────────────────────┘  │ │  │
│  └─────────────────────────────────────────────────────────────┘ │  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. API Group and Version
- **Group**: `discover.k8s.webcenter.fr` (following elasticsearch-operator naming convention)
- **Version**: `v1` (first version for elasticsearch-operator)

### 2. CRD Types
Three new CRDs:
- **Kafka** (`discover.k8s.webcenter.fr/v1`): Discover Kafka connection info (Strimzi-managed or external)
- **Logstash** (`discover.k8s.webcenter.fr/v1`): Discover Logstash connection info (managed or external)
- **Elasticsearch** (`discover.k8s.webcenter.fr/v1`): Discover Elasticsearch connection info (managed or external)

### 3. Shared Discover Type
```go
type Discover struct {
    // Secret for file-based data (certs, keys, keystores)
    TargetSecretFileRef *corev1.LocalObjectReference `json:"targetSecretRef,omitempty"`
    // Default: {name}-file

    // Secret for env-based data (URLs, passwords)
    TargetSecretEnvRef *corev1.LocalObjectReference `json:"targetSecretEnvRef,omitempty"`
    // Default: {name}-env

    // Logical name for env var suffix and mount path
    Name *string `json:"name,omitempty"`
    // Default: resource name
}

type DiscoverStatus struct {
    SecretFileRef *string `json:"secretFileRef,omitempty"`
    SecretEnvRef  *string `json:"secretEnvRef,omitempty"`
}

type DiscoverRef struct {
    Kafka         *corev1.LocalObjectReference `json:"kafka,omitempty"`
    Logstash      *corev1.LocalObjectReference `json:"logstash,omitempty"`
    Elasticsearch *corev1.LocalObjectReference `json:"elasticsearch,omitempty"`
}
```

### 4. Secret Structure
Each Discover CR generates **two secrets**:

| Secret | Default Name | Purpose | Type Annotation |
|--------|-------------|---------|-----------------|
| **Env Secret** | `{name}-env` | Environment variables (connection details, passwords) | `discover.k8s.webcenter.fr/type: "env"` |
| **File Secret** | `{name}-file` | Certificates, keystores, truststores | `discover.k8s.webcenter.fr/type: "file"` |

### 5. Secret Labels and Annotations

**Labels:**
```yaml
labels:
  discoverName: <discover-resource-name>
  discover.k8s.webcenter.fr: "true"
  kafka.discover.k8s.webcenter.fr: "true"      # Type-specific
  # OR logstash.discover.k8s.webcenter.fr: "true"
  # OR elasticsearch.discover.k8s.webcenter.fr: "true"
```

**Annotations:**
```yaml
annotations:
  discover.k8s.webcenter.fr: "true"
  discover.k8s.webcenter.fr/mountPath: <internal-name>
  discover.k8s.webcenter.fr/type: "env" | "file"
  kafka.discover.k8s.webcenter.fr: "true"  # Type-specific
```

### 6. Environment Variable Naming

| Prefix | Variables |
|--------|-----------|
| `KAFKA_` | `KAFKA_BOOTSTRAP_SERVERS_{SUFFIX}`, `KAFKA_AUTH_TYPE_{SUFFIX}`, `KAFKA_USER_PASSWORD_{SUFFIX}`, `KAFKA_CA_TRUSTSTORE_PASSWORD_{SUFFIX}`, etc. |
| `LOGSTASH_` | `LOGSTASH_HOSTS_{SUFFIX}`, `LOGSTASH_PORT_{SUFFIX}`, etc. |
| `ELASTICSEARCH_` | `ELASTICSEARCH_HOSTS_{SUFFIX}`, `ELASTICSEARCH_USERNAME_{SUFFIX}`, `ELASTICSEARCH_PASSWORD_{SUFFIX}`, etc. |

Where `{SUFFIX}` = `strings.ToUpper(strings.ReplaceAll(discover.GetInternalName(), "-", "_"))`

### 7. File Secret Contents

| Discover Type | File Keys |
|--------------|-----------|
| **Kafka** | `ca.crt`, `ca.p12`, `user.crt`, `user.key`, `user.p12` |
| **Logstash** | `ca.crt`, `user.crt`, `user.key`, `user.p12` |
| **Elasticsearch** | `ca.crt` |

### 8. Workload Integration

**Filebeat** (output only):
- Add `discoverRef []*DiscoverRef` field
- Add `discoverOutputName *string` field (specifies which discover to use as output)
- Auto-generate `output.kafka` or `output.logstash` config from discover secrets
- Mount discover file secrets at `/usr/share/filebeat/discover/{name}/`
- Inject discover env secrets via `envFrom`

**Logstash** (input/output):
- Add `discoverRef []*DiscoverRef` field
- Mount discover file secrets at `/usr/share/logstash/discover/{name}/`
- Inject discover env secrets via `envFrom`
- User manually references env vars and files in pipeline configs

**Metricbeat** (output only):
- Add `discoverRef []*DiscoverRef` field
- Add `discoverOutputName *string` field
- Similar to Filebeat but for Elasticsearch output

### 9. Backward Compatibility

- Existing `elasticsearchRef` and `logstashRef` fields remain functional
- Marked as **deprecated** in favor of `discoverRef`
- Webhook validation allows both patterns (but not both for the same output)
- Migration path: Users can gradually migrate from static refs to Discover

### 10. Strimzi Integration (Kafka)

- **Dependency**: Add `github.com/RedHatInsights/strimzi-client-go v0.40.0` to go.mod
- **Watches**: Strimzi `Kafka` and `KafkaUser` CRs
- **Capability detection**: Check if Strimzi CRDs are installed (`kubeCapability.HasStrimzi`)
- **Listener selection**: Support `targetListener` field to select specific Kafka listener
- **Auth types**: Detect `tls`, `oauth`, `scram-sha-512`, `plain` from listener config

### 11. PKCS#12 Generation

- **Dependency**: Add `software.sslmate.com/src/go-pkcs12 v0.6.0` to go.mod
- **Keystore generation**: Generate `user.p12` from user cert + key
- **Truststore generation**: Generate `ca.p12` from CA certificates
- **Password generation**: Random passwords for keystore/truststore

## Implementation Tasks

### Phase 1: Foundation (Week 1)

1. **Add dependencies to go.mod**
   - `github.com/RedHatInsights/strimzi-client-go v0.40.0`
   - `software.sslmate.com/src/go-pkcs12 v0.6.0`
   - Run `go mod tidy`

2. **Create Discover API package structure**
   - Create `api/discover/v1/` directory
   - Create `groupversion_info.go` with group `discover.k8s.webcenter.fr`
   - Create `zz_generated.deepcopy.go` (run `make generate`)

3. **Create shared Discover types**
   - `api/discover/v1/discover_type.go`: `Discover`, `DiscoverStatus`, `DiscoverRef`, `DiscoverType`
   - `api/discover/v1/discover_func.go`: Helper methods (`GetType()`, `GetName()`)

4. **Create DiscoverObject interface**
   - `pkg/object/discover.go`: `DiscoverObject` and `DiscoverObjectStatus` interfaces

### Phase 2: CRD Definitions (Week 1-2)

5. **Create Kafka Discover CRD**
   - `api/discover/v1/kafka_types.go`: `KafkaSpec`, `KafkaStatus`, `KafkaRef`, `KafkaManagedRef`, `KafkaExternalRef`
   - `api/discover/v1/kafka_func.go`: Helper methods (`GetInternalName()`, `IsManaged()`, `IsExternal()`, `ValidateField()`)
   - `api/discover/v1/kafka_indexer.go`: Field indexers for reverse lookups
   - `api/discover/v1/kafka_webhook.go`: Validation webhook

6. **Create Logstash Discover CRD**
   - `api/discover/v1/logstash_types.go`: `LogstashSpec`, `LogstashStatus`, `LogstashRef`, `LogstashManagedRef`, `LogstashExternalRef`
   - `api/discover/v1/logstash_func.go`: Helper methods
   - `api/discover/v1/logstash_indexer.go`: Field indexers
   - `api/discover/v1/logstash_webhook.go`: Validation webhook

7. **Create Elasticsearch Discover CRD**
   - `api/discover/v1/elasticsearch_types.go`: `ElasticsearchSpec`, `ElasticsearchStatus`, `ElasticsearchRef`
   - `api/discover/v1/elasticsearch_func.go`: Helper methods
   - `api/discover/v1/elasticsearch_indexer.go`: Field indexers
   - `api/discover/v1/elasticsearch_webhook.go`: Validation webhook

8. **Generate CRDs and RBAC**
   - Run `make manifests` to generate CRD YAML files
   - Run `make generate` to generate deepcopy methods
   - Verify CRDs in `config/crd/bases/`

### Phase 3: Discover Controllers (Week 2-3)

9. **Create Discover controller package structure**
   - Create `internal/controller/discover/` directory
   - Create `internal/controller/discover/discover.go`: Core functions (`ReadDiscoversSecrets`, `ComputeDiscoverPod`, `IsDiscoverSecret`, etc.)
   - Create `internal/controller/discover/discover_watcher.go`: `WatchDiscoverSecret` generic watcher

10. **Create Kafka Discover controller**
    - `internal/controller/discover/kafka/kafka_controller.go`: Main controller with `SetupWithManager`
    - `internal/controller/discover/kafka/kafka_watcher.go`: Watch Strimzi Kafka, KafkaUser, Secrets
    - `internal/controller/discover/kafka/secret_kafka_builder.go`: Build Kafka secrets (env + file)
    - `internal/controller/discover/kafka/secret_kafka_reconciler.go`: Reconcile Kafka secrets
    - `internal/controller/discover/kafka/helper.go`: Helper functions (`GetSecretNameForEnv`, `GetSecretNameForFile`, etc.)

11. **Create Logstash Discover controller**
    - `internal/controller/discover/logstash/logstash_controller.go`
    - `internal/controller/discover/logstash/logstash_watcher.go`
    - `internal/controller/discover/logstash/secret_logstash_builder.go`
    - `internal/controller/discover/logstash/secret_logstash_reconciler.go`
    - `internal/controller/discover/logstash/helper.go`

12. **Create Elasticsearch Discover controller**
    - `internal/controller/discover/elasticsearch/elasticsearch_controller.go`
    - `internal/controller/discover/elasticsearch/elasticsearch_watcher.go`
    - `internal/controller/discover/elasticsearch/secret_elasticsearch_builder.go`
    - `internal/controller/discover/elasticsearch/secret_elasticsearch_reconciler.go`
    - `internal/controller/discover/elasticsearch/helper.go`

### Phase 4: Workload Integration (Week 3-4)

13. **Update Filebeat types**
    - Add `DiscoverRef []*discovercrd.DiscoverRef` to `FilebeatSpec`
    - Add `DiscoverOutputName *string` to `FilebeatSpec`
    - Add `SearchDiscoverOutputRef()` method to `Filebeat`
    - Update `filebeat_indexer.go` to index `spec.discover.name`
    - Update `filebeat_webhook.go` to validate discover refs

14. **Update Filebeat controller**
    - Update `filebeat_controller.go`: Add `Watches` for discover secrets
    - Update `filebeat_watcher.go`: Add `discover.WatchDiscoverSecret` call
    - Update `configmap_builder.go`: Auto-generate output config from discover
    - Update `configmap_reconciler.go`: Read discover output secrets
    - Update `statefulset_builder.go`: Call `discover.ComputeDiscoverPod`
    - Update `statefulset_reconciler.go`: Call `discover.ReadDiscoversSecrets`

15. **Update Logstash types**
    - Add `DiscoverRef []*discovercrd.DiscoverRef` to `LogstashSpec`
    - Update `logstash_indexer.go` to index `spec.discover.name`
    - Update `logstash_webhook.go` to validate discover refs

16. **Update Logstash controller**
    - Update `logstash_controller.go`: Add `Watches` for discover secrets
    - Update `logstash_watcher.go`: Add `discover.WatchDiscoverSecret` call
    - Update `statefulset_builder.go`: Call `discover.ComputeDiscoverPod`
    - Update `statefulset_reconciler.go`: Call `discover.ReadDiscoversSecrets`

17. **Update Metricbeat types**
    - Add `DiscoverRef []*discovercrd.DiscoverRef` to `MetricbeatSpec`
    - Add `DiscoverOutputName *string` to `MetricbeatSpec`
    - Update `metricbeat_indexer.go` to index `spec.discover.name`

18. **Update Metricbeat controller**
    - Update `metricbeat_controller.go`: Add `Watches` for discover secrets
    - Update `metricbeat_watcher.go`: Add `discover.WatchDiscoverSecret` call
    - Update `configmap_builder.go`: Auto-generate output config from discover
    - Update `statefulset_builder.go`: Call `discover.ComputeDiscoverPod`
    - Update `statefulset_reconciler.go`: Call `discover.ReadDiscoversSecrets`

### Phase 5: Registration and Integration (Week 4)

19. **Register Discover indexers in main.go**
    - Add `discovercrd.SetupKafkaIndexer`
    - Add `discovercrd.SetupLogstashIndexer`
    - Add `discovercrd.SetupElasticsearchIndexer`

20. **Register Discover webhooks in main.go**
    - Add `discovercrd.SetupKafkaWebhookWithManager`
    - Add `discovercrd.SetupLogstashWebhookWithManager`
    - Add `discovercrd.SetupElasticsearchWebhookWithManager`

21. **Register Discover controllers in main.go**
    - Add `kafkaDiscoverController.SetupWithManager(mgr)`
    - Add `logstashDiscoverController.SetupWithManager(mgr)`
    - Add `elasticsearchDiscoverController.SetupWithManager(mgr)`

22. **Add Strimzi capability detection**
    - Update `internal/controller/common/kubernetes.go`: Add `HasStrimzi` field to `KubernetesCapability`
    - Update capability detection logic in `cmd/main.go`

### Phase 6: Testing (Week 4-5)

23. **Unit tests for Discover types**
    - Test `DiscoverRef.GetType()` and `DiscoverRef.GetName()`
    - Test validation functions

24. **Unit tests for Discover controllers**
    - Test secret builders (Kafka, Logstash, Elasticsearch)
    - Test secret reconcilers
    - Test watchers

25. **Unit tests for workload integration**
    - Test `ComputeDiscoverPod` function
    - Test `ReadDiscoversSecrets` function
    - Test Filebeat config generation with discover

26. **Integration tests**
    - Test end-to-end flow: Create Discover CR → Verify secrets → Create workload → Verify pod
    - Test with Strimzi Kafka (if available in test environment)
    - Test certificate rotation

27. **E2E tests**
    - Test Filebeat → Kafka via Discover
    - Test Filebeat → Logstash via Discover
    - Test Logstash → Elasticsearch via Discover
    - Test Logstash → Kafka via Discover

### Phase 7: Documentation (Week 5)

28. **Create documentation**
    - `documentations/discover/kafka.md`: Kafka Discover usage
    - `documentations/discover/logstash.md`: Logstash Discover usage
    - `documentations/discover/elasticsearch.md`: Elasticsearch Discover usage
    - `documentations/filebeat/discover-settings.md`: Filebeat with Discover
    - `documentations/logstash/discover-settings.md`: Logstash with Discover
    - `documentations/metricbeat/discover-settings.md`: Metricbeat with Discover

29. **Create example manifests**
    - `config/samples/discover_v1_kafka.yaml`
    - `config/samples/discover_v1_logstash.yaml`
    - `config/samples/discover_v1_elasticsearch.yaml`
    - `config/samples/beat_v1_filebeat_discover.yaml`
    - `config/samples/logstash_v1_discover.yaml`

30. **Update existing documentation**
    - Mark `elasticsearchRef` and `logstashRef` as deprecated
    - Add migration guide from static refs to Discover

## File Structure

```
elasticsearch-operator/
├── api/
│   └── discover/
│       └── v1/
│           ├── groupversion_info.go
│           ├── discover_type.go          # Shared Discover types
│           ├── discover_func.go          # Shared helper functions
│           ├── kafka_types.go            # Kafka Discover CRD
│           ├── kafka_func.go             # Kafka helper functions
│           ├── kafka_indexer.go          # Kafka field indexers
│           ├── kafka_webhook.go          # Kafka validation webhook
│           ├── logstash_types.go         # Logstash Discover CRD
│           ├── logstash_func.go          # Logstash helper functions
│           ├── logstash_indexer.go       # Logstash field indexers
│           ├── logstash_webhook.go       # Logstash validation webhook
│           ├── elasticsearch_types.go    # Elasticsearch Discover CRD
│           ├── elasticsearch_func.go     # Elasticsearch helper functions
│           ├── elasticsearch_indexer.go  # Elasticsearch field indexers
│           ├── elasticsearch_webhook.go  # Elasticsearch validation webhook
│           └── zz_generated.deepcopy.go  # Generated
├── internal/
│   └── controller/
│       └── discover/
│           ├── discover.go               # Core discover functions
│           ├── discover_watcher.go       # Generic discover secret watcher
│           ├── kafka/
│           │   ├── kafka_controller.go
│           │   ├── kafka_watcher.go
│           │   ├── secret_kafka_builder.go
│           │   ├── secret_kafka_reconciler.go
│           │   └── helper.go
│           ├── logstash/
│           │   ├── logstash_controller.go
│           │   ├── logstash_watcher.go
│           │   ├── secret_logstash_builder.go
│           │   ├── secret_logstash_reconciler.go
│           │   └── helper.go
│           └── elasticsearch/
│               ├── elasticsearch_controller.go
│               ├── elasticsearch_watcher.go
│               ├── secret_elasticsearch_builder.go
│               ├── secret_elasticsearch_reconciler.go
│               └── helper.go
├── pkg/
│   └── object/
│       └── discover.go                   # DiscoverObject interface
├── config/
│   ├── crd/
│   │   └── bases/
│   │       ├── discover.k8s.webcenter.fr_kafkas.yaml
│   │       ├── discover.k8s.webcenter.fr_logstashes.yaml
│   │       └── discover.k8s.webcenter.fr_elasticsearches.yaml
│   └── samples/
│       ├── discover_v1_kafka.yaml
│       ├── discover_v1_logstash.yaml
│       ├── discover_v1_elasticsearch.yaml
│       ├── beat_v1_filebeat_discover.yaml
│       └── logstash_v1_discover.yaml
└── documentations/
    ├── discover/
    │   ├── kafka.md
    │   ├── logstash.md
    │   └── elasticsearch.md
    ├── filebeat/
    │   └── discover-settings.md
    ├── logstash/
    │   └── discover-settings.md
    └── metricbeat/
        └── discover-settings.md
```

## Key Implementation Details

### Discover Core Functions (internal/controller/discover/discover.go)

```go
// IsDiscoverSecret checks if a secret is a discover secret
func IsDiscoverSecret(s *corev1.Secret) bool {
    return s.Annotations["discover.k8s.webcenter.fr"] == "true"
}

// IsDiscoverSecretEnv checks if a secret is a discover env secret
func IsDiscoverSecretEnv(s *corev1.Secret) bool {
    return IsDiscoverSecret(s) && s.Annotations["discover.k8s.webcenter.fr/type"] == "env"
}

// IsDiscoverSecretFile checks if a secret is a discover file secret
func IsDiscoverSecretFile(s *corev1.Secret) bool {
    return IsDiscoverSecret(s) && s.Annotations["discover.k8s.webcenter.fr/type"] == "file"
}

// GetDiscoverMountPathFromAnnotations gets the mount path from secret annotations
func GetDiscoverMountPathFromAnnotations(s *corev1.Secret) string {
    return s.Annotations["discover.k8s.webcenter.fr/mountPath"]
}

// ReadDiscoversSecrets reads all secrets for referenced discover CRs
func ReadDiscoversSecrets(ctx context.Context, c client.Client, logger *logrus.Entry,
    o client.Object, discoversRef []*discovercrd.DiscoverRef) ([]*corev1.Secret, *reconcile.Result, error) {
    // Implementation: Read each Discover CR, check status, read env and file secrets
}

// ComputeDiscoverPod injects discover secrets into pod template
func ComputeDiscoverPod(podTemplate k8sbuilder.PodTemplateBuilder,
    container k8sbuilder.ContainerBuilder, secrets []*corev1.Secret, mountBasePath string) {
    // Implementation: Mount file secrets as volumes, inject env secrets via envFrom
}
```

### Kafka Secret Builder Key Logic

```go
func buildKafkaSecrets(kk *discovercrd.Kafka, kafkaCluster *strimzicrd.Kafka,
    secretCaKafka *corev1.Secret, secretUserKafka *corev1.Secret,
    secretCustomCaKafka *corev1.Secret) ([]*corev1.Secret, error) {
    // 1. Collect CA certificates (custom + Strimzi cluster CA)
    // 2. Determine bootstrap URL from Strimzi Kafka listeners
    // 3. Detect auth type from listener configuration
    // 4. Copy user cert/key from Strimzi KafkaUser secret or external secret
    // 5. Generate PKCS#12 keystore (user.p12) from user cert + key
    // 6. Generate PKCS#12 truststore (ca.p12) from CA certificates
    // 7. Generate random passwords for keystore/truststore
    // 8. Build env secret with KAFKA_* variables
    // 9. Build file secret with certificates
}
```

### Filebeat Config Auto-Generation

```go
// In configmap_builder.go
switch discoverOutputType {
case discovercrd.DiscoverTypeKafka:
    // Find KAFKA_BOOTSTRAP_SERVERS_* env var
    // Generate output.kafka section with SSL config
    filebeatConf["output.kafka"] = map[string]any{
        "enabled":     true,
        "client_id":   "${POD_NAME}",
        "hosts":       []string{"${KAFKA_BOOTSTRAP_SERVERS_KAFKA}"},
        "ssl": map[string]any{
            "enabled": true,
            "certificate_authorities": []string{
                "/usr/share/filebeat/discover/kafka/ca.crt"
            },
            "certificate": "/usr/share/filebeat/discover/kafka/user.crt",
            "key":         "/usr/share/filebeat/discover/kafka/user.key",
            "verification_mode": "full",
        },
    }
case discovercrd.DiscoverTypeLogstash:
    // Similar for Logstash output
}
```

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Strimzi dependency adds complexity** | High | Make Strimzi optional via capability detection; Kafka Discover only works if Strimzi is installed |
| **PKCS#12 generation for Java workloads** | Medium | Use `software.sslmate.com/src/go-pkcs12` library; test with Logstash |
| **Backward compatibility break** | High | Keep existing `elasticsearchRef`/`logstashRef` working; mark as deprecated; provide migration guide |
| **Secret rotation complexity** | Medium | Use existing multiphase reconciler pattern; implement proper diff logic to avoid infinite loops |
| **Cross-namespace references** | Medium | Use `fullname` field selectors (namespace/name) for all indexers |
| **Certificate format differences** | Medium | Support both PEM and PKCS#12; generate both formats in file secrets |

## Validation Plan

### Unit Tests
- [ ] Test all Discover type helper functions
- [ ] Test secret builders for each Discover type
- [ ] Test secret reconcilers (create, update, delete)
- [ ] Test watchers (field selector queries)
- [ ] Test `ComputeDiscoverPod` function
- [ ] Test `ReadDiscoversSecrets` function
- [ ] Test Filebeat config generation with discover

### Integration Tests
- [ ] Test Discover CR creation and secret generation
- [ ] Test workload creation with discoverRef
- [ ] Test pod spec generation (volumes, envFrom, volumeMounts)
- [ ] Test secret rotation (update Discover CR → verify secrets updated → verify pod restart)
- [ ] Test with Strimzi Kafka (if available)

### E2E Tests
- [ ] Filebeat → Kafka via Discover (with Strimzi)
- [ ] Filebeat → Logstash via Discover
- [ ] Logstash → Elasticsearch via Discover
- [ ] Logstash → Kafka via Discover
- [ ] Metricbeat → Elasticsearch via Discover
- [ ] Certificate rotation end-to-end

## Migration Path

### For Existing Users

1. **Phase 1**: Continue using `elasticsearchRef`/`logstashRef` (no changes required)
2. **Phase 2**: Create Discover CRs alongside existing workloads
3. **Phase 3**: Update workloads to use `discoverRef` instead of `elasticsearchRef`/`logstashRef`
4. **Phase 4**: Remove deprecated `elasticsearchRef`/`logstashRef` fields (future release)

### Example Migration

**Before (static ref):**
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
```

**After (discover):**
```yaml
# Step 1: Create Discover CR
apiVersion: discover.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: my-elasticsearch-discover
spec:
  elasticsearchRef:
    managed:
      name: my-elasticsearch
---
# Step 2: Update Filebeat to use discover
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
spec:
  discoverRef:
    - elasticsearch:
        name: my-elasticsearch-discover
  discoverOutputName: my-elasticsearch-discover
```

## Open Questions

1. **Should Metricbeat support Discover?** 
   - Recommendation: Yes, for consistency with Filebeat
   - Metricbeat would use Discover for Elasticsearch output

2. **Should Kibana support Discover?**
   - Recommendation: No, Kibana already has `elasticsearchRef` which works well
   - Can be added later if needed

3. **Should we support multiple outputs for Filebeat?**
   - Current OpenSearch implementation: Only one output via `discoverOutputName`
   - Recommendation: Keep single output for now, can be extended later

4. **PKCS#12 password storage**
   - Recommendation: Store passwords in env secret (not file secret)
   - File secret contains only binary keystore/truststore files

## Success Criteria

- [ ] All three Discover CRDs (Kafka, Logstash, Elasticsearch) are functional
- [ ] Filebeat can output to Kafka and Logstash via Discover
- [ ] Logstash can input from Kafka and output to Elasticsearch via Discover
- [ ] Metricbeat can output to Elasticsearch via Discover
- [ ] Strimzi integration works for managed Kafka clusters
- [ ] External Kafka/Logstash/Elasticsearch clusters are supported
- [ ] Certificate rotation works (secrets updated → pods restart)
- [ ] Backward compatibility maintained (existing refs still work)
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Documentation is complete

## References

- OpenSearch operator Discover implementation: `/projects/opensearch-operator-k8s/`
- Strimzi client: `github.com/RedHatInsights/strimzi-client-go`
- PKCS#12 library: `software.sslmate.com/src/go-pkcs12`
- operator-sdk-extra multiphase reconciler: `github.com/disaster37/operator-sdk-extra/v3`
