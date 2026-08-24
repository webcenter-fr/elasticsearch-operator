# Cerebro Controller Migration Plan

Migrate the `cerebro` controller in `github.com/webcenter-fr/elasticsearch-operator` so it
deploys the **new upstream cerebro application** (`github.com/disaster37/cerebro`, a Go/Hertz +
Vue3/Nuxt4 rewrite) instead of the legacy Scala/Play `lmenezes/cerebro`.

---

## 0. Critical context for the implementer — read this first

The task brief assumed that `github.com/disaster37/cerebro` is a **Kubernetes operator** that
"defines the modern cerebro operator API types (`api/v1`, `controllers/`, `internal/controller/`
with a `Cerebro` CRD and `Host` component model)". **This is factually wrong**, and the plan below
is built on what the upstream source actually contains. Verified facts from `/projects/cerebro`:

- `go.mod` declares `module cerebro` (go 1.26). Dependencies are `cloudwego/hertz`, `spf13/viper`,
  `urfave/cli/v2`, `modernc.org/sqlite`, `go-ldap/ldap`, `elastic/go-elasticsearch/v9`,
  `disaster37/opensearch/v4`, `gorilla/securecookie`.
- There is **no** `sigs.k8s.io/controller-runtime`, **no** `operator-sdk`, **no** kubebuilder.
  A repo-wide grep for `kubebuilder|controller-runtime|Reconcile|CustomResourceDefinition|operator-sdk`
  returns **zero matches**.
- Layout is an application: `cmd/cerebro/main.go`, `internal/{config,domain,handler,service,repository,server,auth}`,
  `app/` (Nuxt frontend), `conf/application.yaml`, `deploy/{docker,helm,kubernetes}`.
- Latest release is **v0.1.0** (2026-08-24): image `ghcr.io/disaster37/cerebro:v0.1.0`,
  Helm chart `oci://ghcr.io/disaster37/cerebro-charts/cerebro --version 0.1.0`. Other tags: `dev`,
  `dev-<pr>`. **There is no `latest` tag.**

Consequences that shape this whole plan:

1. **Do NOT add `github.com/disaster37/cerebro` to `go.mod`.** Its module path is `cerebro`, it is
   not importable, and it exports no API types to reuse.
2. The `Cerebro` and `Host` CRDs are **this operator's own API**. They are not "upstream types" to
   be replaced. They stay, with only the minimal edits listed in §3.
3. The real migration is a **workload/config migration**: new image, new config file format
   (HOCON → YAML), new file paths, new env var names, new probes, new CLI args.

`/projects/cerebro/deploy/kubernetes/*` and `/projects/cerebro/conf/application.yaml` are the
authoritative references for how the new app must be deployed and configured.

---

## 1. Objective and scope

### Objective

Rewrite the cerebro controller so that a `Cerebro` custom resource produces a working deployment of
the **new** cerebro application, keeping the existing CRD surface and multiphase reconciler
architecture intact.

### In scope

- `api/cerebro/v1/cerebro_types.go`: change `Version` default, remove the obsolete `Node` field.
- `internal/controller/cerebro/helper.go`: new default image + default version constant.
- `internal/controller/cerebro/configmap_builder.go`: generate `application.yaml` (YAML) instead of
  `application.conf` (HOCON).
- `internal/controller/cerebro/deployment_builder.go`: new args, env, volumes, mounts, probes.
- `internal/controller/cerebro/secret_application_builder.go` +
  `secret_application_reconciler.go`: rename secret data key to `session-key`, with in-place value
  migration.
- All affected `testdata/*.yml` golden files and `*_test.go` files.
- `config/samples/cerebro_v1_cerebro.yaml`: bump `version`.
- Regenerated `config/crd/bases/cerebro.k8s.webcenter.fr_cerebroes.yaml` and `bundle/`.

### Out of scope (explicitly)

- **No new `go.mod` dependency** on upstream cerebro (see §0).
- **No new CRD API version** and **no conversion webhook**. Group/kind/version
  (`cerebro.k8s.webcenter.fr/v1`, kinds `Cerebro`, `Host`) are unchanged, therefore the `PROJECT`
  file is **not** modified.
- **No PVC / persistence.** Decision: keep `emptyDir` for the SQLite data path. REST request history
  remains ephemeral across pod restarts, matching today's behavior. (Upstream's own manifests use a
  RWO PVC with `replicas: 1` + `strategy: Recreate`; adopting that is a deliberate follow-up.)
- **No `Host` CRD expansion.** The new app supports per-host `type` (`elasticsearch`/`opensearch`),
  `auth` (username/password) and `headers-whitelist`. We do not add these fields now. Omitting
  `type` is safe: upstream auto-detects the flavor via `GET /` sniffing `version.distribution`
  (see `/projects/cerebro/internal/config/hosts.go` and `CHANGES.md`).
- No changes to `config/rbac/*` (no new resource kinds are managed).
- No changes to the `Host` webhook, indexers, or watchers.

---

## 2. What actually changes (old app → new app)

Derived by diffing the current builders against `/projects/cerebro/conf/application.yaml`,
`/projects/cerebro/deploy/docker/Dockerfile` and `/projects/cerebro/deploy/kubernetes/deployment.yaml`.

| Concern | Current (legacy Scala) | New (upstream Go) |
|---|---|---|
| Image | `lmenezes/cerebro` | `ghcr.io/disaster37/cerebro` |
| Default version | `latest` | `v0.1.0` |
| Config file name | `application.conf` | `application.yaml` |
| Config format | HOCON, `${?ENV}` substitution | YAML, Viper env binding |
| Config mount dir | `/etc/cerebro` | `/opt/cerebro/conf` |
| CLI args | `-Dconfig.file=/etc/cerebro/application.conf` | `--config /opt/cerebro/conf/application.yaml` |
| Session secret env | `APPLICATION_SECRET` | `CEREBRO_SECRET` |
| Secret data key | `application` | `session-key` |
| Data dir | `/var/db/cerebro` (`data.path=/var/db/cerebro/cerebro.db`) | `/data` (`data.path=/data/cerebro.db`) |
| Data volume name | `db` | `data` |
| Logs volume | `/opt/cerebro/logs` (emptyDir) | **removed** (logs to stdout) |
| `/tmp` volume | present | kept (required, `readOnlyRootFilesystem: true`) |
| Liveness | TCP 9000 | `HTTP GET /favicon.ico` |
| Readiness | `HTTP GET /` | `HTTP GET /favicon.ico` |
| Startup | TCP 9000 | TCP 9000 (kept) |
| Port | 9000 | 9000 (`CEREBRO_PORT` override exists; unused) |
| `spec.deployment.node` | Node.js process opts | **removed** (obsolete: Go binary) |
| Extra-config merge key | `extraConfigs["application.conf"]` | `extraConfigs["application.yaml"]` |
| `spec.config` semantics | raw HOCON appended | YAML document deep-merged |

Env vars **retained** because Viper binds them explicitly
(`/projects/cerebro/internal/config/config.go`, `bindings` map): `AUTH_TYPE`, `BASIC_AUTH_USER`,
`BASIC_AUTH_PWD`, `LDAP_URL`, `LDAP_BASE_DN`, `LDAP_METHOD`, `LDAP_USER_TEMPLATE`, `LDAP_BIND_DN`,
`LDAP_BIND_PWD`, `LDAP_GROUP_BASE_DN`, `LDAP_USER_ATTR`, `LDAP_USER_ATTR_TEMPLATE`, `LDAP_GROUP`.
Users still set these via `spec.deployment.env` / `envFrom`; the controller does not need to inject
them. Because Viper precedence is `env > config file`, the generated YAML can safely ship empty
defaults for all auth settings — this is why the `${?ENV}` placeholder trick is no longer needed.

---

## 3. Files to create / modify / delete

### Modify

| Path | Change |
|---|---|
| `api/cerebro/v1/cerebro_types.go` | `Version` default `latest` → `v0.1.0`; delete `CerebroDeploymentSpec.Node`; update `Config`/`ExtraConfigs` doc comments to say YAML |
| `internal/controller/cerebro/helper.go` | `defaultImage`, new `defaultVersion`, rewrite `GetContainerImage` |
| `internal/controller/cerebro/configmap_builder.go` | full rewrite → YAML generation |
| `internal/controller/cerebro/deployment_builder.go` | args, env, volumes, mounts, probes |
| `internal/controller/cerebro/secret_application_builder.go` | data key `application` → `session-key` |
| `internal/controller/cerebro/secret_application_reconciler.go` | in-place key migration logic (§6.2) |
| `config/samples/cerebro_v1_cerebro.yaml` | `version: 0.9.4` → `version: v0.1.0` |
| `config/crd/bases/cerebro.k8s.webcenter.fr_cerebroes.yaml` | regenerated (`make manifests`) — do not hand-edit |
| `bundle/manifests/cerebro.k8s.webcenter.fr_cerebroes.yaml` | regenerated (`make bundle`) — do not hand-edit |
| `bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml` | regenerated (`make bundle`) |
| `api/cerebro/v1/zz_generated.deepcopy.go` | regenerated (`make generate`) |

### Modify (tests + golden files)

| Path | Change |
|---|---|
| `internal/controller/cerebro/testdata/configmap_default.yml` | `application.conf` → `application.yaml`, new YAML body |
| `internal/controller/cerebro/testdata/configmap_elasticsearch_targets.yml` | same + YAML `hosts:` list |
| `internal/controller/cerebro/testdata/configmap_elasticsearch_external_targets.yml` | same + YAML `hosts:` list |
| `internal/controller/cerebro/testdata/deployment_default.yml` | image/args/env/volumes/probes |
| `internal/controller/cerebro/testdata/deployment_default_openshift.yml` | same |
| `internal/controller/cerebro/testdata/deployment_complet.yml` | same + new checksum values |
| `internal/controller/cerebro/helper_test.go` | `TestGetContainerImage` expectations |
| `internal/controller/cerebro/configmap_builder_test.go` | extra-config key → `application.yaml` |
| `internal/controller/cerebro/deployment_builder_test.go` | `Version: "8.5.1"` → `"v0.1.0"` |
| `internal/controller/cerebro/cerebro_controller_test.go` | `Version`, and `application.yaml` + YAML host assertions |

### Delete

| Path | Reason |
|---|---|
| `internal/controller/cerebro/testdata/deployment_default openshift.yml` | Stray duplicate with a **space** in the filename; unreferenced by any test (only `deployment_default_openshift.yml` is used). Remove as cleanup. |

### Not modified (call out to avoid churn)

`PROJECT`, `go.mod`, `go.sum`, `Makefile`, `Dockerfile`, `config/rbac/*`, `config/crd/kustomization.yaml`,
`api/cerebro/v1/host_types.go`, `host_webhook.go`, `host_func.go`, `host_indexer.go`,
`cerebro_indexer.go`, `cerebro_func.go`, `cerebro_watcher.go`, `cerebro_controller.go`,
and all `*_reconciler.go` except `secret_application_reconciler.go`.

---

## 4. Data structures

### 4.1 `api/cerebro/v1/cerebro_types.go`

Only two functional edits. Full resulting type definitions:

```go
const (
	CerebroAnnotationKey = "cerebro.k8s.webcenter.fr"
)

// CerebroSpec defines the desired state of Cerebro
// +k8s:openapi-gen=true
type CerebroSpec struct {
	shared.ImageSpec `json:",inline"`

	// Version is the Cerebro version to use
	// It correspond to the image tag of ghcr.io/disaster37/cerebro
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=v0.1.0
	Version string `json:"version,omitempty"`

	// Endpoint permit to set endpoints to access on Cerebro from external kubernetes
	// You can set ingress and / or load balancer
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Endpoint shared.EndpointSpec `json:"endpoint,omitempty"`

	// Config is the Cerebro config as a YAML document.
	// It is deep merged over the config generated by the operator, and wins on conflict.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config *string `json:"config,omitempty"`

	// ExtraConfigs is extra config files stored on the config directory.
	// The key is the file name and the value is the file contend.
	// The special key `application.yaml` is deep merged over the generated config
	// instead of being written as a standalone file.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExtraConfigs map[string]string `json:"extraConfigs,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Deployment CerebroDeploymentSpec `json:"deployment,omitempty"`
}

// CerebroDeploymentSpec permit to set the deployment settings
type CerebroDeploymentSpec struct {
	shared.Deployment `json:",inline"`
}

// CerebroStatus defines the observed state of Cerebro
type CerebroStatus struct {
	multiphase.DefaultMultiPhaseObjectStatus `json:",inline"`

	// Url is the Cerebro endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Url string `json:"url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Cerebro is the Schema for the cerebroes API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Deployment,apps/v1}}
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Cerebro struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CerebroSpec   `json:"spec,omitempty"`
	Status CerebroStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CerebroList contains a list of Cerebro
type CerebroList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cerebro `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cerebro{}, &CerebroList{})
}
```

> `CerebroDeploymentSpec` now only embeds `shared.Deployment`. Keep the named wrapper type — it is
> referenced by tests and by `cerebro_indexer.go`; collapsing it into `shared.Deployment` would be a
> gratuitous API churn.

### 4.2 `Host` types — unchanged

`api/cerebro/v1/host_types.go` is **not** modified. For reference, the shape the config builder
consumes:

```go
type HostSpec struct {
	CerebroRef       HostCerebroRef   `json:"cerebroRef"`
	ElasticsearchRef ElasticsearchRef `json:"elasticsearchRef"`
}

type HostCerebroRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ElasticsearchRef struct {
	ManagedElasticsearchRef  *corev1.LocalObjectReference `json:"managed,omitempty"`
	ExternalElasticsearchRef *ElasticsearchExternalRef    `json:"external,omitempty"`
}

type ElasticsearchExternalRef struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}
```

### 4.3 New internal config structs (`configmap_builder.go`, not API types)

These are **unexported, non-CRD** helper structs used only to marshal the generated
`application.yaml`. They mirror `/projects/cerebro/internal/config/config.go` key names exactly
(Viper reads `secret`, `basePath`, `data.path`, `rest.history.size`, `es.gzip`, `auth.*`, `hosts`).

```go
type cerebroConfig struct {
	Secret   string             `yaml:"secret"`
	BasePath string             `yaml:"basePath"`
	Data     cerebroDataConfig  `yaml:"data"`
	Rest     cerebroRestConfig  `yaml:"rest"`
	Es       cerebroEsConfig    `yaml:"es"`
	Auth     cerebroAuthConfig  `yaml:"auth"`
	Hosts    []cerebroHostEntry `yaml:"hosts"`
}

type cerebroDataConfig struct {
	Path string `yaml:"path"`
}

type cerebroRestConfig struct {
	History cerebroRestHistoryConfig `yaml:"history"`
}

type cerebroRestHistoryConfig struct {
	Size int `yaml:"size"`
}

type cerebroEsConfig struct {
	Gzip bool `yaml:"gzip"`
}

type cerebroAuthConfig struct {
	Type     string                    `yaml:"type"`
	Settings cerebroAuthSettingsConfig `yaml:"settings"`
}

type cerebroAuthSettingsConfig struct {
	Username     string                   `yaml:"username"`
	Password     string                   `yaml:"password"`
	URL          string                   `yaml:"url"`
	BaseDN       string                   `yaml:"base-dn"`
	Method       string                   `yaml:"method"`
	UserTemplate string                   `yaml:"user-template"`
	BindDN       string                   `yaml:"bind-dn"`
	BindPW       string                   `yaml:"bind-pw"`
	GroupSearch  cerebroGroupSearchConfig `yaml:"group-search"`
}

type cerebroGroupSearchConfig struct {
	BaseDN           string `yaml:"base-dn"`
	UserAttr         string `yaml:"user-attr"`
	UserAttrTemplate string `yaml:"user-attr-template"`
	Group            string `yaml:"group"`
}

// cerebroHostEntry is one entry of the `hosts` list.
// `type` is intentionally omitted: upstream auto-detects elasticsearch vs opensearch
// via `GET /` when the field is empty.
type cerebroHostEntry struct {
	Host string `yaml:"host"`
	Name string `yaml:"name"`
}
```

### 4.4 Target generated `application.yaml`

With no hosts (defaults mirror `/projects/cerebro/conf/application.yaml`, but with `secret` empty
because it is supplied via `CEREBRO_SECRET`):

```yaml
auth:
  settings:
    base-dn: ""
    bind-dn: ""
    bind-pw: ""
    group-search:
      base-dn: ""
      group: ""
      user-attr: ""
      user-attr-template: ""
    method: simple
    password: ""
    url: ""
    user-template: uid=%s,%s
    username: ""
  type: ""
basePath: /
data:
  path: /data/cerebro.db
es:
  gzip: true
hosts: []
rest:
  history:
    size: 50
secret: ""
```

With hosts, `hosts:` becomes:

```yaml
hosts:
- host: https://es1-es.default.svc:9200
  name: es1
- host: https://test1.domain.local
  name: test1
```

> Keys are alphabetically sorted because the builder always round-trips through
> `map[string]any` before marshalling (see §5.2). This ordering is deterministic and is what the
> golden files must contain. Generate the golden files by running the tests once and copying actual
> output — do not hand-write them.

---

## 5. Function signatures and implementation detail

### 5.1 `internal/controller/cerebro/helper.go`

```go
const (
	defaultImage   = "ghcr.io/disaster37/cerebro"
	defaultVersion = "v0.1.0"

	// legacyVersion is the value the old CRD defaulted `spec.version` to. Existing
	// Cerebro resources have it persisted, and no `latest` tag exists upstream, so it
	// must be remapped to avoid ImagePullBackOff on upgrade.
	legacyVersion = "latest"
)

// GetConfigMapName permit to get the configMap name that store the config
func GetConfigMapName(cb *cerebrocrd.Cerebro) (configMapName string)

// GetSecretNameForApplication permit to get the secret name for application
func GetSecretNameForApplication(cb *cerebrocrd.Cerebro) (secretName string)

// GetServiceName permit to get the service name
func GetServiceName(cb *cerebrocrd.Cerebro) (serviceName string)

// GetLoadBalancerName permit to get the load balancer name
func GetLoadBalancerName(cb *cerebrocrd.Cerebro) (serviceName string)

// GetIngressName permit to get the ingress name
func GetIngressName(cb *cerebrocrd.Cerebro) (ingressName string)

// GetDeploymentName permit to get the deployement name
func GetDeploymentName(cb *cerebrocrd.Cerebro) (name string)

// GetContainerImage permit to get the image name
func GetContainerImage(cb *cerebrocrd.Cerebro) string

// IsLegacyVersion returns true when spec.version still carries the obsolete
// `latest` value inherited from the previous CRD default.
func IsLegacyVersion(cb *cerebrocrd.Cerebro) bool

func getLabels(cb *cerebrocrd.Cerebro, customLabels ...map[string]string) (labels map[string]string)
func getAnnotations(cb *cerebrocrd.Cerebro, customAnnotation ...map[string]string) (annotations map[string]string)
```

All the `Get*Name` helpers keep their current bodies and suffixes (`-config-cb`,
`-application-cb`, `-cb`, `-lb-cb`). Do **not** rename resources: renaming would orphan every
existing ConfigMap/Service/Deployment.

New bodies:

```go
// GetContainerImage permit to get the image name
func GetContainerImage(cb *cerebrocrd.Cerebro) string {
	version := defaultVersion
	if cb.Spec.Version != "" && cb.Spec.Version != legacyVersion {
		version = cb.Spec.Version
	}

	image := defaultImage
	if cb.Spec.Image != "" {
		image = cb.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

// IsLegacyVersion returns true when spec.version still carries the obsolete
// `latest` value inherited from the previous CRD default.
func IsLegacyVersion(cb *cerebrocrd.Cerebro) bool {
	return cb.Spec.Version == legacyVersion
}
```

> `legacyVersion` remap applies **only** when `spec.image` is not overridden? No — it applies
> unconditionally to the *tag*. A user who sets a custom `spec.image` mirror plus
> `version: latest` also gets `v0.1.0`. That is intentional and safer than emitting a
> non-existent tag; users wanting `latest` must now state a real tag.

### 5.2 `internal/controller/cerebro/configmap_builder.go`

Signature is **unchanged** so `configmap_reconciler.go` needs no edit:

```go
func buildConfigMaps(
	cb *cerebrocrd.Cerebro,
	esList []elasticsearchcrd.Elasticsearch,
	externalList []cerebrocrd.ElasticsearchExternalRef,
) (configMaps []*corev1.ConfigMap, err error)
```

Add two private helpers:

```go
// buildCerebroConfig computes the base config from the Cerebro spec and its enrolled hosts.
func buildCerebroConfig(
	esList []elasticsearchcrd.Elasticsearch,
	externalList []cerebrocrd.ElasticsearchExternalRef,
) *cerebroConfig

// renderCerebroConfig marshals the base config, then deep merges the user supplied
// YAML overlays on top of it. Overlays win on conflict. Later overlays win over earlier ones.
func renderCerebroConfig(base *cerebroConfig, overlays ...string) (string, error)
```

Implementation notes:

- `buildCerebroConfig` constants: `BasePath: "/"`, `Data.Path: "/data/cerebro.db"`,
  `Rest.History.Size: 50`, `Es.Gzip: true`, `Auth.Settings.Method: "simple"`,
  `Auth.Settings.UserTemplate: "uid=%s,%s"`, `Secret: ""`, `Auth.Type: ""`.
- `Hosts` must be a non-nil empty slice (`make([]cerebroHostEntry, 0, ...)`) so it marshals as
  `hosts: []` rather than `hosts: null`.
- Managed host name: `es.Name`, overridden by `es.Spec.ClusterName` when non-empty (preserve current
  logic). Address: `elasticsearchcontrollers.GetPublicUrl(&es, "", false)` (preserve current call,
  keeps the existing import of the elasticsearch controllers package).
- External host: `externalRef.Name` / `externalRef.Address`.
- Ordering: managed hosts first (in `esList` order), then external hosts. This matches the current
  builder and keeps golden files stable.

`renderCerebroConfig` algorithm — **always** round-trip through a map, even with no overlays, so key
ordering is identical in both cases:

```go
func renderCerebroConfig(base *cerebroConfig, overlays ...string) (string, error) {
	raw, err := yaml.Marshal(base)
	if err != nil {
		return "", errors.Wrap(err, "Error when marshal cerebro config")
	}

	merged := map[string]any{}
	if err = yaml.Unmarshal(raw, &merged); err != nil {
		return "", errors.Wrap(err, "Error when normalize cerebro config")
	}

	for _, overlay := range overlays {
		if overlay == "" {
			continue
		}
		src := map[string]any{}
		if err = yaml.Unmarshal([]byte(overlay), &src); err != nil {
			return "", errors.Wrap(err, "Error when parse provided cerebro config, it must be a valid YAML document")
		}
		if err = mergo.Merge(&merged, src, mergo.WithOverride); err != nil {
			return "", errors.Wrap(err, "Error when merge provided config with default config")
		}
	}

	out, err := yaml.Marshal(merged)
	if err != nil {
		return "", errors.Wrap(err, "Error when marshal merged cerebro config")
	}

	return string(out), nil
}
```

Imports: `gopkg.in/yaml.v3` and `dario.cat/mergo` — both already direct dependencies in `go.mod`
(`gopkg.in/yaml.v3 v3.0.1`, `dario.cat/mergo v1.0.2`). No `go.mod` change.

`buildConfigMaps` body:

```go
func buildConfigMaps(cb *cerebrocrd.Cerebro, esList []elasticsearchcrd.Elasticsearch, externalList []cerebrocrd.ElasticsearchExternalRef) (configMaps []*corev1.ConfigMap, err error) {
	configMaps = make([]*corev1.ConfigMap, 0, 1)

	overlays := make([]string, 0, 2)
	if cb.Spec.Config != nil && *cb.Spec.Config != "" {
		overlays = append(overlays, *cb.Spec.Config)
	}
	if cb.Spec.ExtraConfigs["application.yaml"] != "" {
		overlays = append(overlays, cb.Spec.ExtraConfigs["application.yaml"])
	}

	config, err := renderCerebroConfig(buildCerebroConfig(esList, externalList), overlays...)
	if err != nil {
		return nil, err
	}

	expectedConfig := map[string]string{
		"application.yaml": config,
	}

	// Extra config files are added as standalone keys. mergo does not override the
	// already computed `application.yaml`, which was consumed as an overlay above.
	if err = mergo.Merge(&expectedConfig, cb.Spec.ExtraConfigs); err != nil {
		return nil, errors.Wrap(err, "Error when merge provided config with default config")
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cb.Namespace,
			Name:        GetConfigMapName(cb),
			Labels:      getLabels(cb),
			Annotations: getAnnotations(cb),
		},
		Data: expectedConfig,
	}

	configMaps = append(configMaps, configMap)

	return configMaps, nil
}
```

Remove the `strings` and `fmt` imports if they become unused.

### 5.3 `internal/controller/cerebro/deployment_builder.go`

Signature unchanged:

```go
func buildDeployments(
	cerebro *cerebrocrd.Cerebro,
	secretsChecksum []*corev1.Secret,
	configMapsChecksum []*corev1.ConfigMap,
	isOpenshift bool,
) (dpls []*appv1.Deployment, err error)

func getContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container)
```

Edits, in the order they appear in the current file:

1. **Args** — replace
   ```go
   cb.Container().Args = []string{"-Dconfig.file=/etc/cerebro/application.conf"}
   ```
   with
   ```go
   cb.Container().Args = []string{
   	"--config",
   	"/opt/cerebro/conf/application.yaml",
   }
   ```
   (Explicit `--config` rather than relying on the `conf/application.yaml` default, so the
   deployment does not silently depend on the image's `WORKDIR /opt/cerebro`.)

2. **Env** — rename the injected secret env var and key:
   ```go
   {
   	Name: "CEREBRO_SECRET",
   	ValueFrom: &corev1.EnvVarSource{
   		SecretKeyRef: &corev1.SecretKeySelector{
   			LocalObjectReference: corev1.LocalObjectReference{
   				Name: GetSecretNameForApplication(cerebro),
   			},
   			Key: "session-key",
   		},
   	},
   },
   ```
   Keep `NODE_NAME`, `NAMESPACE`, `POD_NAME`, `POD_IP` (downward-API vars, harmless and useful for
   users' own templating).

3. **Volume mounts** — replace the four-entry list with:
   ```go
   cb.WithVolumeMount([]corev1.VolumeMount{
   	{
   		Name:      "config",
   		MountPath: "/opt/cerebro/conf",
   	},
   	{
   		Name:      "data",
   		MountPath: "/data",
   	},
   	{
   		Name:      "tmp",
   		MountPath: "/tmp",
   	},
   }, k8sbuilder.Merge)
   ```

4. **Liveness probe** — TCP → HTTP:
   ```go
   cb.WithLivenessProbe(&corev1.Probe{
   	TimeoutSeconds:   5,
   	PeriodSeconds:    30,
   	FailureThreshold: 3,
   	SuccessThreshold: 1,
   	ProbeHandler: corev1.ProbeHandler{
   		HTTPGet: &corev1.HTTPGetAction{
   			Path:   "/favicon.ico",
   			Port:   intstr.FromInt(9000),
   			Scheme: corev1.URISchemeHTTP,
   		},
   	},
   }, k8sbuilder.OverwriteIfDefaultValue)
   ```

5. **Readiness probe** — path `/` → `/favicon.ico` (everything else unchanged).

6. **Startup probe** — unchanged (TCP 9000).

7. **Pod volumes** — replace `db`/`logs`/`tmp` with `data`/`tmp`:
   ```go
   ptb.WithVolumes([]corev1.Volume{
   	{
   		Name: "config",
   		VolumeSource: corev1.VolumeSource{
   			ConfigMap: &corev1.ConfigMapVolumeSource{
   				LocalObjectReference: corev1.LocalObjectReference{
   					Name: GetConfigMapName(cerebro),
   				},
   			},
   		},
   	},
   	{
   		Name: "data",
   		VolumeSource: corev1.VolumeSource{
   			EmptyDir: &corev1.EmptyDirVolumeSource{},
   		},
   	},
   	{
   		Name: "tmp",
   		VolumeSource: corev1.VolumeSource{
   			EmptyDir: &corev1.EmptyDirVolumeSource{},
   		},
   	},
   }, k8sbuilder.Merge)
   ```

Everything else in this file is unchanged: checksum annotations, container name `cerebro`, port
`http`/9000, resources, image/pull-policy, security contexts (non-OpenShift pins UID/GID 1000 which
matches the upstream image's `cerebro` user; OpenShift variant omits them), pod security context
`fsGroup: 1000`, `terminationGracePeriodSeconds: 30`, labels/annotations, selector, replicas.

### 5.4 `internal/controller/cerebro/secret_application_builder.go`

```go
const applicationSecretKey = "session-key"

// buildApplicationSecrets permit to build credential secret
func buildApplicationSecrets(o *cerebrocrd.Cerebro) (secrets []*corev1.Secret, err error)
```

Body: identical to today except `Data` uses `applicationSecretKey` instead of `"application"`.
Keep `password.Generate(64, 10, 0, false, true)`.

### 5.5 `internal/controller/cerebro/secret_application_reconciler.go`

Signature unchanged:

```go
func (r *applicationSecretReconciler) Read(
	ctx context.Context,
	o *cerebrocrd.Cerebro,
	data map[string]any,
	logger *logrus.Entry,
) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error)
```

Replace the current blanket carry-over:

```go
// Never update existing credentials
if currentApplicationSecret != nil {
	expectedApplicationSecrets[0].Data = currentApplicationSecret.Data
}
```

with key-aware preservation + migration:

```go
// Never regenerate existing credentials.
// Legacy secrets carry the value under the `application` key: re-key it to
// `session-key` in place so that already issued session cookies stay valid.
if currentApplicationSecret != nil {
	if v, ok := currentApplicationSecret.Data[applicationSecretKey]; ok {
		expectedApplicationSecrets[0].Data[applicationSecretKey] = v
	} else if v, ok := currentApplicationSecret.Data["application"]; ok {
		logger.Infof("Migrate secret %s from key `application` to key `%s`", GetSecretNameForApplication(o), applicationSecretKey)
		expectedApplicationSecrets[0].Data[applicationSecretKey] = v
	}
}
```

The old `application` key is intentionally **not** copied into the expected object, so the
reconciler converges the secret to a single key.

---

## 6. Step-by-step implementation sequence

Dependency-ordered. Compile after each numbered step.

1. **API types.** Edit `api/cerebro/v1/cerebro_types.go` per §4.1 (`Version` default → `v0.1.0`,
   delete `CerebroDeploymentSpec.Node`, refresh doc comments).
2. **Regenerate deepcopy.** `make generate`. Confirm `api/cerebro/v1/zz_generated.deepcopy.go` still
   compiles (removing a `string` field needs no deepcopy change, but regenerate for hygiene).
3. **Verify no other reader of `.Node`.** `grep -rn 'Spec.Deployment.Node' --include='*.go' .` must
   return only `internal/controller/kibana/deployment_builder.go` (that is Kibana's **own**
   `Node` field on a different type — do not touch it).
4. **Helper.** Apply §5.1 to `internal/controller/cerebro/helper.go`.
5. **Config builder.** Rewrite `configmap_builder.go` per §5.2, adding the structs from §4.3.
6. **Secret builder + reconciler.** Apply §5.4 then §5.5.
7. **Deployment builder.** Apply the seven edits in §5.3.
8. **Compile.** `go build ./...` must pass.
9. **Regenerate CRDs.** `make manifests`. Confirm `spec.version` default is `v0.1.0` and the
   `deployment.node` property is gone from
   `config/crd/bases/cerebro.k8s.webcenter.fr_cerebroes.yaml`.
10. **Sample.** Set `version: v0.1.0` in `config/samples/cerebro_v1_cerebro.yaml`.
11. **Unit tests + golden files.** Update the `*_test.go` inputs (§7.1), run the builder tests, and
    regenerate the six `testdata/*.yml` files from actual output. Delete the stray
    `deployment_default openshift.yml`.
12. **Envtest.** Update `cerebro_controller_test.go` assertions (§7.2) and run the suite.
13. **Full verification.** Run everything in §9.
14. **Bundle.** `make bundle` (regenerates `bundle/manifests/*`). Only if the repo's release flow
    requires it in the same change; otherwise leave to the release step.

---

## 7. Testing plan

### 7.1 Unit tests

**`internal/controller/cerebro/helper_test.go` — `TestGetContainerImage`**

```go
func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &cerebrocrd.Cerebro{ /* name test, ns default */ }
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.1.0", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v0.2.0"
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.2.0", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v0.2.0", GetContainerImage(o))

	// When the legacy `latest` default is still persisted, it is remapped
	o.Spec.Image = ""
	o.Spec.Version = "latest"
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.1.0", GetContainerImage(o))
}
```

Add:

```go
func TestIsLegacyVersion(t *testing.T)
```
covering `""` → false, `"v0.1.0"` → false, `"latest"` → true.

`TestGetConfigMapName`, `TestGetServiceName`, `TestGetLoadBalancerName`, `TestGetIngressName`,
`TestGetDeploymentName`, `TestGetLabels`, `TestGetAnnotations` are unchanged and must still pass —
they are the guard that no resource got renamed.

**`internal/controller/cerebro/configmap_builder_test.go` — `TestBuildConfigMap`**

- Case 1 (no targets): change `Config` to a YAML overlay and the extra-config key:
  ```go
  Config: ptr.To("basePath: /cerebro\n"),
  ExtraConfigs: map[string]string{
      "application.yaml": "rest:\n  history:\n    size: 100\n",
      "log4j.yml":        "log.test: test\n",
  },
  ```
  Golden `testdata/configmap_default.yml` must then show `basePath: /cerebro`,
  `rest.history.size: 100`, and a standalone `log4j.yml` key.
- Case 2 (managed targets) and case 3 (external targets): inputs unchanged, goldens become YAML.

Add a new test asserting overlay precedence and error handling:

```go
func TestBuildConfigMapWithInvalidConfig(t *testing.T) {
	o := &cerebrocrd.Cerebro{ /* ... */ }
	o.Spec.Config = ptr.To("this: [is: not valid yaml")
	_, err := buildConfigMaps(o, nil, nil)
	assert.Error(t, err)
}
```

**`internal/controller/cerebro/deployment_builder_test.go` — `TestBuildDeployment`**

Three existing cases keep their structure. In the "complexe sample" case change
`Version: "8.5.1"` → `Version: "v0.1.0"` and `ExtraConfigs` key `log4j.yaml` may stay. Checksum
annotations in `testdata/deployment_complet.yml` **will change** (config content changed) — copy the
new values from actual output.

**Golden files** — regenerate all six from test output. Each `deployment_*.yml` must show:
`image: ghcr.io/disaster37/cerebro:v0.1.0`, `args: ["--config", "/opt/cerebro/conf/application.yaml"]`,
env `CEREBRO_SECRET` → `secretKeyRef{name: test-application-cb, key: session-key}`, mounts
`/opt/cerebro/conf`, `/data`, `/tmp`, volumes `config`/`data`/`tmp`, HTTP `/favicon.ico` liveness
and readiness, TCP startup. No `logs` volume, no `/var/db/cerebro`.

### 7.2 Envtest (`internal/controller/cerebro/cerebro_controller_test.go`)

Existing suite `TestCerebroControllerSuite` / `TestCerebroController` with steps
`doCreateCerebroStep`, `doUpdateCerebroStep`, `doAddHostStep`, `doDeleteCerebroStep`.

- `doCreateCerebroStep`: `Version: "0.9.4"` → `"v0.1.0"`.
- `doAddHostStep` `Check`: replace the HOCON assertions
  ```go
  assert.Contains(t, cm.Data["application.conf"], fmt.Sprintf("name = \"%s\"", key.Name))
  assert.Contains(t, cm.Data["application.conf"], fmt.Sprintf("host = \"https://%s-es.%s.svc:9200\"", key.Name, key.Namespace))
  ```
  with YAML equivalents
  ```go
  assert.Contains(t, cm.Data["application.yaml"], fmt.Sprintf("name: %s", key.Name))
  assert.Contains(t, cm.Data["application.yaml"], fmt.Sprintf("host: https://%s-es.%s.svc:9200", key.Name, key.Namespace))
  ```
  and change the `GetConfigMapName` lookup's expected data key accordingly.
- All other assertions (secret/service/lb/ingress/route/configmap/deployment exist with
  owner refs, `status.url` non-empty, `status.isOnError` false) are unchanged and must still pass.

Add a step to cover the secret re-key migration — the highest-risk change:

```go
func doMigrateLegacySecretStep() test.TestStep[*cerebrocrd.Cerebro]
```

`Do`: patch the existing `<name>-application-cb` secret so its `Data` contains only
`{"application": []byte("legacy-value")}`, then touch the `Cerebro` spec to trigger reconcile.
`Check`: poll until the secret's `Data` has `session-key == "legacy-value"` and no `application`
key. This asserts both that the value is preserved (sessions survive) and that the secret converges.

### 7.3 Commands

```sh
# unit only, fast loop
go test ./internal/controller/cerebro/... -run 'TestBuildConfigMap|TestBuildDeployment|TestGet|TestIsLegacyVersion' -count 1

# api package
go test ./api/cerebro/... -count 1

# full suite as CI runs it (envtest, serialized)
make test
```

`make test` depends on `manifests generate fmt envtest` and runs
`./api/... ./internal/controller/... ./pkg/...` with `-p 1`, `ES_OPERATOR_ENVTEST=true`,
`-timeout 1200s`.

---

## 8. Edge cases, error handling and validation

### 8.1 `spec.version: latest` on existing resources (highest-impact)

The previous CRD had `+kubebuilder:default=latest`, so the apiserver **already persisted**
`version: latest` into every existing `Cerebro`. Changing the marker default does **not** rewrite
stored objects. Since `ghcr.io/disaster37/cerebro:latest` does not exist, a naive change causes
immediate `ImagePullBackOff` on every upgraded cluster.

Handling: `GetContainerImage` remaps `latest` → `defaultVersion` (§5.1), and the controller emits a
warning so operators know to fix their manifests. Add to `CerebroReconciler.Configure` in
`cerebro_controller.go`:

```go
if IsLegacyVersion(o) {
	h.Recorder().Eventf(o, corev1.EventTypeWarning, "DeprecatedVersion",
		"spec.version=%q is not a valid tag for %s and is treated as %s; set an explicit version",
		legacyVersion, defaultImage, defaultVersion)
}
```

This is the only edit to `cerebro_controller.go`.

### 8.2 Removal of `spec.deployment.node`

Removing the property from the CRD schema means the apiserver **prunes** it from stored objects
(structural schema pruning) — no validation error, no rejection. Users who set it lose a field that
had no effect on a Go binary anyway. No conversion webhook is needed because there is no version
bump. Document in release notes.

### 8.3 Config format break (`spec.config`, `extraConfigs`)

This is a **hard semantic break** and cannot be auto-migrated: HOCON is not YAML.

- A user with HOCON in `spec.config` will now get a YAML parse error from
  `renderCerebroConfig`. That surfaces as a reconcile error → `OnError` → `status.isOnError=true`,
  `status.lastErrorMessage` set, error event recorded, and the multiphase framework requeues. The
  error message must be actionable, hence the wrapped text "it must be a valid YAML document".
- Users must rename `extraConfigs["application.conf"]` → `extraConfigs["application.yaml"]`.
  A leftover `application.conf` key is harmless: it is written as an inert standalone file in the
  ConfigMap and ignored by the app.
- Note that many HOCON snippets are *accidentally* valid YAML (`key = value` parses as a scalar
  string `"key = value"`, actually invalid as a mapping) — do not attempt to be clever; fail loudly.

### 8.4 Secret key rename

Covered by §5.5. Two failure modes avoided:

- **Value regeneration.** If migration simply generated a new `session-key`, all existing session
  cookies would be invalidated (users logged out). Re-keying preserves the value.
- **Never-converging secret.** The pre-existing "never update existing credentials" logic copies
  `current.Data` wholesale; without the key-aware branch the secret would forever contain only
  `application`, `session-key` would never exist, and the container would start with an empty
  `CEREBRO_SECRET`. With `auth.type` unset the app still boots (upstream only hard-fails on an empty
  secret when auth is enabled), so this would be a **silent** security regression rather than a
  crash — which is exactly why the explicit migration branch matters.

Ordering is already correct: `newApplicationSecretReconciler` is the **first** entry in
`stepReconcilers`, so the secret converges before the deployment phase reads it.

### 8.5 Rollout / restart behavior

`deployment_builder.go` stamps SHA256 checksums of the config ConfigMap and the application Secret
into pod annotations. Both change in this migration, so **every** cerebro pod rolls once on
upgrade. Expected and desirable (the new args/mounts require a new pod spec anyway). Default
deployment strategy (RollingUpdate) is retained since we use `emptyDir`, not a RWO PVC — so no
`Recreate`/`replicas: 1` constraint applies. Note that upstream's own manifest pins `replicas: 1`
because SQLite is single-writer; with `emptyDir` each replica gets its own private DB, so multiple
replicas do not corrupt data but give **inconsistent REST history** per pod. Document this; do not
enforce it.

### 8.6 Status, finalizers, orphans — unchanged

- `status.url` computation (`computeCerebroUrl`) is unaffected: still ingress → route →
  loadbalancer → in-cluster service `:9000`, all HTTP/HTTPS on port 9000.
- Readiness gating in `OnSuccess` (compare `dpl.Status.ReadyReplicas` to
  `o.Spec.Deployment.Replicas`, set `Ready` condition, `RunningPhase`/`StartingPhase`,
  `RequeueAfter: 30s`) is unchanged.
- Finalizer `cerebro.k8s.webcenter.fr/finalizer` handling on `Host` objects
  (added/removed in `configmap_reconciler.Read`, cleaned in `CerebroReconciler.Delete`) is unchanged.
- The `Host` validating webhook (`elasticsearchRef` must be managed or external) is unchanged.
- Orphan handling: the `logs` and `db` volumes disappear from the pod spec; they were `emptyDir`,
  so nothing is orphaned in the cluster.

### 8.7 Probe path vs `basePath`

`/favicon.ico` assumes `basePath: /`. A user who overrides `basePath` via `spec.config` will break
both probes (upstream's Dockerfile carries the same caveat). Not solved here — recorded as an open
question in §10.

---

## 9. Verification commands

Run from `/projects/elasticsearch-operator`:

```sh
# 1. code generation is in sync (deepcopy + CRDs/RBAC/webhooks)
make generate
make manifests
git diff --exit-code api/cerebro/v1/zz_generated.deepcopy.go config/crd/bases/

# 2. compile everything
go build ./...

# 3. formatting and static analysis
make fmt
make vet          # == go vet ./...

# 4. targeted unit tests (fast)
go test ./internal/controller/cerebro/... -count 1

# 5. api package tests
go test ./api/cerebro/... -count 1

# 6. full test suite incl. envtest (as CI does)
make test

# 7. CRD applies cleanly against a live cluster (optional, needs kubeconfig)
make install
make install-sample

# 8. OLM bundle regeneration (only if the change ships a release)
make bundle
```

Expected assertions after step 1: `config/crd/bases/cerebro.k8s.webcenter.fr_cerebroes.yaml`
contains `default: v0.1.0` under `spec.properties.version` and has **no** `node` property under
`spec.properties.deployment.properties`.

Notes on tooling:
- There is **no** `.golangci.yml` in this repository, so "lint" is `make vet` only. Do not introduce
  a linter config as part of this change.
- `make test` needs envtest assets; `ENVTEST_K8S_VERSION` is pinned to `1.25.x` in the Makefile.
- `make manifests` also runs `go run ./hack/strip-crd-cel-validations`; expect that post-processing
  to touch the regenerated CRD.

---

## 10. Risks and open questions

### Risks

| # | Risk | Severity | Mitigation |
|---|---|---|---|
| 1 | Existing `version: latest` → `ImagePullBackOff` | High | `legacyVersion` remap + warning event (§8.1) |
| 2 | Secret key rename silently yields empty `CEREBRO_SECRET` | High | Key-aware migration in reconciler + dedicated envtest step (§5.5, §7.2) |
| 3 | HOCON in `spec.config` now fails to parse | Medium | Loud, actionable error; release-note the break (§8.3) |
| 4 | Golden-file churn hides a real regression | Medium | Regenerate from actual output, then **read the diff** before committing |
| 5 | `/favicon.ico` probes break under custom `basePath` | Low | Documented caveat (§8.7) |
| 6 | Multiple replicas give divergent REST history with `emptyDir` | Low | Document; upstream pins `replicas: 1` for the same reason |
| 7 | `v0.1.0` is a brand-new upstream release (first Go/Nuxt release, 2026-08-24) | Medium | Pin the exact tag; do not use `dev`. Expect follow-up bumps |

### Open questions

1. **`basePath` support.** Should `basePath` become a first-class `CerebroSpec` field so the
   controller can derive probe paths and the ingress path together, instead of users smuggling it
   through `spec.config`? Recommend deferring, but it is the natural next increment.
2. **Persistence.** Deliberately deferred. If REST-history durability is wanted, add an optional
   `persistence` field (reuse `shared.DeploymentPersistenceSpec`), create a PVC, force
   `replicas: 1` + `strategy: Recreate`, and add `persistentvolumeclaims` RBAC.
3. **Host flavor / auth / header whitelist.** The new app supports per-host `type`
   (`elasticsearch`|`opensearch`), `auth.username`/`auth.password` and `headers-whitelist`. Adding
   these to `HostSpec` would let the operator target OpenSearch clusters explicitly and authenticate
   to secured external clusters. Currently relying on auto-detection and anonymous access to
   external clusters.
4. **Elasticsearch credentials.** The managed-cluster path emits only a URL. If the managed ES has
   security enabled, cerebro cannot authenticate. The legacy controller had the same gap, so this is
   pre-existing, not a regression — but it is now easy to fix via the upstream per-host `auth` field
   (ties into Q3).
5. **Envtest version pin.** `ENVTEST_K8S_VERSION = 1.25.x` while `k8s.io/*` libs are at `v0.36.4`.
   Likely a latent mismatch unrelated to this change; flag to the maintainers rather than fix here.
6. **Bundle regeneration timing.** Confirm whether `bundle/` should be regenerated in this change or
   only at release (`make release`). The plan assumes release-time.
