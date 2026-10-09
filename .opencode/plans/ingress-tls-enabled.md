# Plan: Align ingress/route status URL scheme with `tlsEnabled` helpers + `TlsSpec.Enabled` default true

## 1. Goal & context

**User request (verbatim):** "look on project /projects/opensearch-operator how tls is enable for ingress. Is base on ingress.tlsEnabled: true instead the secretRef set. Implement the same way here."

Note: `/projects/opensearch-operator` does not exist. The correct reference operator is
`/projects/opensearch-operator-k8s`. It is the upstream equivalent of this operator and already
computes the status URL scheme from the `tlsEnabled` helpers.

**Two workstreams in this plan:**
- **Workstream A (primary):** fix the status URL scheme in the three controllers so it derives from
  the `IsTlsEnabled()` helpers (matching the ingress/route builders and the reference operator).
- **Workstream B (revision):** make `shared.TlsSpec.Enabled` default to `true` end-to-end — the Go
  doc comment currently says "Default to false" while the kubebuilder marker and runtime behavior are
  already `true`. This is a documentation/schema consistency fix.

**Workstream A — problem.** In this operator the status URL scheme (the `scheme://` prefix reported
in `Status.Url`) is derived from the *presence of `SecretRef`* (ingress) or a *raw pointer check*
(route), which is inconsistent with how the ingress/route are actually built. The builders already
decide whether to attach a TLS block using the `IsTlsEnabled()` helpers (defaulting to `true`), so
the reported URL can be `http://` while the object actually serves TLS, and vice-versa.

**Reference implementation (verified):**
- `/projects/opensearch-operator-k8s/internal/controller/opensearch/opensearch_controller.go`
  `computeOpensearchUrl` (line 359): ingress scheme uses
  `es.Spec.Endpoint.Ingress.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled()`; route scheme uses
  `es.Spec.Endpoint.Route.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled()`.
- `/projects/opensearch-operator-k8s/internal/controller/dashboard/dashboard_controller.go`
  `computeDashboardUrl` (line 287): same pattern with `kb`.
- `/projects/opensearch-operator-k8s/internal/controller/cerebro/cerebro_controller.go`
  `computeCerebroUrl` (line 270): ingress uses `cb.Spec.Endpoint.Ingress.IsTlsEnabled()`; route uses
  `cb.Spec.Endpoint.Route.IsTlsEnabled()` (Cerebro has no `Spec.Tls`).

**Target (this operator) — the three diverging functions:**
- `internal/controller/elasticsearch/elasticsearch_controller.go` `computeElasticsearchUrl` (line 366)
- `internal/controller/kibana/kibana_controller.go` `computeKibanaUrl` (line 292)
- `internal/controller/cerebro/cerebro_controller.go` `computeCerebroUrl` (line 276)

These are the only three `compute*Url` functions in the repo (logstash/filebeat/metricbeat do not
set `Status.Url`), so the scope is exactly these three controllers — matching the reference.

**Helper semantics (identical in both projects, verified):**
- `shared.EndpointIngressSpec.IsTlsEnabled()` → `*TlsEnabled` if non-nil, else `true`.
- `shared.EndpointRouteSpec.IsTlsEnabled()` → `*TlsEnabled` if non-nil, else `true`.
- `shared.TlsSpec.IsTlsEnabled()` → `false` only when `Enabled != nil && !*Enabled`, else `true`.

**Builders already correct (verified, DO NOT touch):** the ingress/route builders already attach the
TLS block using exactly the same `IsTlsEnabled()` helper combination:
- `elasticsearch/ingress_builder.go:55` and `elasticsearch/route_builder.go:67`:
  `es.Spec.Tls.IsTlsEnabled() || es.Spec.Endpoint.Ingress.IsTlsEnabled()` (and `...Route...`).
- `kibana/ingress_builder.go:40` and `kibana/route_builder.go:49`: same with `kb`.
- `cerebro/ingress_builder.go:33` and `cerebro/route_builder.go:48`: `cb.Spec.Endpoint.Ingress.IsTlsEnabled()` / `...Route.IsTlsEnabled()`.

Therefore the status URL scheme must use the *same boolean expression* the builder uses, so the
reported scheme always matches what is actually served.

## 2. Files to change — Workstream A (status URL scheme)

Only the `if` condition inside the ingress and route branches changes. The `url = ...` assignment
line, the `if/else` braces, and the `scheme = "https"/"http"` bodies are unchanged. Go indentation is
a single tab per level; the snippets below render it as spaces but the coder must preserve tabs
exactly (run `gofmt` afterward).

### 2.1 `internal/controller/elasticsearch/elasticsearch_controller.go`

Function `computeElasticsearchUrl` (line 366).

**Ingress branch** (currently lines 372–379):

Before:
```go
	if es.IsIngressEnabled() {
		url = es.Spec.Endpoint.Ingress.Host

		if es.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
```
After:
```go
	if es.IsIngressEnabled() {
		url = es.Spec.Endpoint.Ingress.Host

		if es.Spec.Endpoint.Ingress.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
```

**Route branch** (currently lines 380–387):

Before:
```go
	} else if es.IsRouteEnabled() {
		url = es.Spec.Endpoint.Route.Host

		if es.Spec.Endpoint.Route.TlsEnabled != nil && *es.Spec.Endpoint.Route.TlsEnabled {
			scheme = "https"
		} else {
			scheme = "http"
		}
```
After:
```go
	} else if es.IsRouteEnabled() {
		url = es.Spec.Endpoint.Route.Host

		if es.Spec.Endpoint.Route.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
```

### 2.2 `internal/controller/kibana/kibana_controller.go`

Function `computeKibanaUrl` (line 292).

**Ingress branch** (currently lines 313–320):

Before:
```go
	if kb.Spec.Endpoint.IsIngressEnabled() {
		url = kb.Spec.Endpoint.Ingress.Host

		if kb.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
```
After:
```go
	if kb.Spec.Endpoint.IsIngressEnabled() {
		url = kb.Spec.Endpoint.Ingress.Host

		if kb.Spec.Endpoint.Ingress.IsTlsEnabled() || kb.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
```

**Route branch** (currently lines 321–329):

Before:
```go
	} else if kb.Spec.Endpoint.IsRouteEnabled() {
		url = kb.Spec.Endpoint.Route.Host

		if kb.Spec.Endpoint.Route.TlsEnabled != nil && *kb.Spec.Endpoint.Route.TlsEnabled {
			scheme = "https"
		} else {
			scheme = "http"
		}

```
After:
```go
	} else if kb.Spec.Endpoint.IsRouteEnabled() {
		url = kb.Spec.Endpoint.Route.Host

		if kb.Spec.Endpoint.Route.IsTlsEnabled() || kb.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}

```

### 2.3 `internal/controller/cerebro/cerebro_controller.go`

Function `computeCerebroUrl` (line 276). Cerebro has no `Spec.Tls`, so the scheme depends only on
the endpoint helper (matching the reference exactly).

**Ingress branch** (currently lines 282–289):

Before:
```go
	if cb.Spec.Endpoint.IsIngressEnabled() {
		url = cb.Spec.Endpoint.Ingress.Host

		if cb.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
```
After:
```go
	if cb.Spec.Endpoint.IsIngressEnabled() {
		url = cb.Spec.Endpoint.Ingress.Host

		if cb.Spec.Endpoint.Ingress.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
```

**Route branch** (currently lines 290–297):

Before:
```go
	} else if cb.Spec.Endpoint.IsRouteEnabled() {
		url = cb.Spec.Endpoint.Route.Host

		if cb.Spec.Endpoint.Route.TlsEnabled != nil && *cb.Spec.Endpoint.Route.TlsEnabled {
			scheme = "https"
		} else {
			scheme = "http"
		}
```
After:
```go
	} else if cb.Spec.Endpoint.IsRouteEnabled() {
		url = cb.Spec.Endpoint.Route.Host

		if cb.Spec.Endpoint.Route.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
```

The load-balancer and internal-service branches in all three functions are **unchanged**
(they already match the reference; the reference leaves them as-is too).

## 3. Files to change — Workstream B (`TlsSpec.Enabled` default true)

### 3.1 Source fix

`api/shared/tls.go` (field `Enabled`, currently lines 6–12). Change only the comment line:

Before:
```go
	// Enabled permit to enabled TLS
	// Default to false
```
After:
```go
	// Enabled permit to enabled TLS
	// Default to true
```

Keep the surrounding markers unchanged (`+operator-sdk:csv:customresourcedefinitions:type=spec`,
`+optional`, `+kubebuilder:default=true`). Do **not** add or alter any `+kubebuilder:default` marker:
every generated CRD that exposes `tls.enabled` already carries `default: true` (verified in the four
CRD files below), so no `default: true` needs to be added anywhere.

### 3.2 Regeneration strategy (preferred when tools/network are available)

1. `make manifests` — runs `controller-gen` and regenerates `config/crd/bases/*`. Expected diff:
   only the `tls.enabled` `description:` block in the two base CRDs flips `Default to false` →
   `Default to true`; the `default: true` line is unchanged; no other fields change.
2. `make bundle` — runs `operator-sdk generate kustomize manifests` then
   `kustomize build config/manifests | operator-sdk generate bundle`, regenerating the base CSV, the
   bundle CRDs and the bundle CSV. Expected diff: only the `tls.enabled` description text flips
   `Default to false` → `Default to true` in the affected descriptors.

Expected generated files whose **only** change should be the `tls.enabled` description:
- `config/crd/bases/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml`
- `config/crd/bases/kibana.k8s.webcenter.fr_kibanas.yaml`
- `bundle/manifests/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml`
- `bundle/manifests/kibana.k8s.webcenter.fr_kibanas.yaml`
- `config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml`
- `bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml`

⚠️ **Regeneration risk — must verify.** `make bundle` additionally runs `yq` version-bump edits
(`.spec.version` → `0.0.49`, `.spec.replaces` → `...0.0.48`, `.metadata.name`) and fully rewrites the
base CSV from the current Go types. The current base CSV still contains **stale `v1alpha1` owned-CRD
entries** (the api tree now only has `v1`; verified no `v1alpha1` package remains), so a full
`operator-sdk generate kustomize manifests` may drop/rewrite those legacy entries and produce a much
larger diff than intended. **Rule:** after running `make bundle`, inspect `git diff`; if anything
beyond the `tls.enabled` description text changed (version bump, dropped/rewritten `v1alpha1` blocks,
or any other descriptor), **revert the generated files and use the manual-edit fallback in §3.3**.
Do not commit a broad regeneration as part of this narrow fix.

### 3.3 Manual-edit fallback (deterministic, no tooling required)

Change `Default to false` → `Default to true` at exactly these locations. Do **not** global
find-and-replace — many unrelated fields in these files also contain "Default to false"; edit only the
listed lines (disambiguated by the `enabled:`/`default: true`/`tls.enabled` context).

| # | File | Line(s) | Current text | New text |
|---|------|---------|--------------|----------|
| 1 | `api/shared/tls.go` | 8 | `	// Default to false` | `	// Default to true` |
| 2 | `config/crd/bases/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml` | 23487 | `                      Default to false` (immediately after `Enabled permit to enabled TLS`, inside the `enabled:` block that has `default: true` above it) | `                      Default to true` |
| 3 | `config/crd/bases/kibana.k8s.webcenter.fr_kibanas.yaml` | 10391 | same context | `                      Default to true` |
| 4 | `bundle/manifests/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml` | 6078 | same context | `                      Default to true` |
| 5 | `bundle/manifests/kibana.k8s.webcenter.fr_kibanas.yaml` | 1847 | same context | `                      Default to true` |
| 6 | `config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml` | 1109 | `      - description: Enabled permit to enabled TLS on Kibana Default to false` | `      - description: Enabled permit to enabled TLS on Kibana Default to true` |
| 7 | `config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml` | 2673 | `          Default to false` (block scalar under `description: |-` + `Enabled permit to enabled TLS`, followed by `path: tls.enabled`) | `          Default to true` |
| 8 | `config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml` | 3424 | same context | `          Default to true` |
| 9 | `bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml` | 1271 | same context | `          Default to true` |
| 10 | `bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml` | 2022 | same context | `          Default to true` |

**Already correct — no change:** `config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml`
line 469 already reads `Enabled permit to enabled TLS on API Default true` (legacy `v1alpha1`
Elasticsearch descriptor). Leave it as-is.

### 3.4 Runtime behavior is unchanged

This workstream is a documentation/schema consistency fix only:
- `shared.TlsSpec.IsTlsEnabled()` already returns `true` when `Enabled` is nil (`api/shared/tls.go:88-93`),
  and `api/shared/tls_test.go:34-51` already asserts `TlsSpec{}.IsTlsEnabled() == true`.
- Every generated CRD already declares `default: true` for `tls.enabled` (verified in the four CRD
  files listed above).
- No Go logic, no controller, and no builder is changed; only the comment string and the generated
  description text are corrected. No new unit test is required for this workstream.

## 4. Rationale (Workstream A)

- **Consistency between builder and status.** The ingress/route builders attach TLS whenever
  `Spec.Tls.IsTlsEnabled() || Endpoint.<Ingress|Route>.IsTlsEnabled()` (ES/Kibana) or
  `Endpoint.<Ingress|Route>.IsTlsEnabled()` (Cerebro). The status URL scheme must evaluate the exact
  same expression, otherwise the reported URL can claim plain HTTP while the ingress/route terminates
  TLS (or vice-versa).
- **`SecretRef` is the wrong signal.** `SecretRef` only selects *which* certificate the ingress/route
  uses; it does not turn TLS on or off. When unset the builder still emits the TLS block (using the
  cluster default certificate) because the helpers default to `true`. Using `SecretRef != nil` to
  decide the scheme produces `http://` for a TLS-serving ingress.
- **Route pointer check misses the default.** `EndpointRouteSpec.IsTlsEnabled()` defaults to `true`
  when `TlsEnabled` is nil, but the current code checks `TlsEnabled != nil && *TlsEnabled`, producing
  `http://` when the field is omitted even though the route serves TLS by default.
- **Helper default semantics.** `IsTlsEnabled()` returning `true` on nil is intentional (documented in
  the helpers) and mirrors the builders, so using the helpers is the correct, single source of truth.

## 5. Edge-case matrix (Workstream A)

Scheme after the change = `endpoint.IsTlsEnabled() || Spec.Tls.IsTlsEnabled()` (ES/Kibana) or
`endpoint.IsTlsEnabled()` (Cerebro). "Before" reflects the current buggy behavior. `SecretRef` never
affects the scheme after the change; before the change it forced `https` on the **ingress** branch
only (route never consulted `SecretRef`).

### Elasticsearch & Kibana (both have `Spec.Tls`)

| # | endpoint | endpoint.TlsEnabled | Spec.Tls.Enabled | SecretRef | Before | After |
|---|----------|---------------------|------------------|-----------|--------|-------|
| 1 | ingress | nil (default true) | nil (default true) | unset | http | **https** |
| 2 | ingress | nil (default true) | nil (default true) | set | https | https |
| 3 | ingress | nil (default true) | true | unset | http | **https** |
| 4 | ingress | nil (default true) | false | unset | http | **https** (ingress still edge-terminates TLS) |
| 5 | ingress | nil (default true) | false | set | https | **https** |
| 6 | ingress | true | nil (default true) | unset | http | **https** |
| 7 | ingress | true | false | unset | http | **https** |
| 8 | ingress | false | nil (default true) | unset | http | **https** (backend TLS) |
| 9 | ingress | false | true | unset | http | **https** |
| 10 | ingress | false | false | unset | http | http |
| 11 | ingress | false | false | set | https | **http** (correct: no TLS block) |
| 12 | route | nil (default true) | nil (default true) | unset | http | **https** |
| 13 | route | nil (default true) | false | unset | http | **https** (route edge-terminates) |
| 14 | route | true | false | unset | https | https |
| 15 | route | true | false | set | https | https |
| 16 | route | false | nil (default true) | unset | http | **https** (backend TLS) |
| 17 | route | false | true | unset | http | **https** |
| 18 | route | false | false | unset | http | http |
| 19 | route | false | false | set | http | http |

### Cerebro (no `Spec.Tls`)

| # | endpoint | endpoint.TlsEnabled | SecretRef | Before | After |
|---|----------|---------------------|-----------|--------|-------|
| 20 | ingress | nil (default true) | unset | http | **https** |
| 21 | ingress | nil (default true) | set | https | https |
| 22 | ingress | true | unset | http | **https** |
| 23 | ingress | true | set | https | https |
| 24 | ingress | false | unset | http | http |
| 25 | ingress | false | set | https | **http** |
| 26 | route | nil (default true) | unset | http | **https** |
| 27 | route | true | unset | https | https |
| 28 | route | false | unset | http | http |

## 6. Tests to add (Workstream A)

Add one new table-driven unit test per package. These are plain `go test` tests (no envtest) placed
in the same package as the envtest suite. They coexist with the envtest suite because Go runs every
`TestXxx` function in a package; the envtest harness is only started by the `suite.Run(...)` call
inside the existing `*_controller_test.go` / `suite_test.go`, while these new tests use only in-memory
structs. There is no build-tag or env-var gating needed (but note: running
`go test ./internal/controller/<pkg>/...` does start the envtest suite; see §8 for how to target the
unit tests specifically).

**Zero-value reconciler safety (verified).** Each `compute*Url` method calls `h.Client()` **only** in
the load-balancer branch. The ingress, route, and internal-service branches never touch the client, so
constructing `&ElasticsearchReconciler{}` / `&KibanaReconciler{}` / `&CerebroReconciler{}` (zero value)
and calling the unexported method is safe for the ingress/route cases below. Do not add any test-only
production hooks.

### 6.1 `internal/controller/elasticsearch/compute_url_test.go` (new, `package elasticsearch`)

Function `TestComputeElasticsearchUrl(t *testing.T)`.

Imports: `context`, `testing`, `github.com/stretchr/testify/assert`,
`elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"`,
`"github.com/webcenter-fr/elasticsearch-operator/api/shared"`,
`v1 "k8s.io/api/core/v1"`, `"k8s.io/utils/ptr"`.

Use a slice of cases `{name string; ingress, route bool; ingressTLS, routeTLS, backendTLS *bool; secretRef *v1.LocalObjectReference; want string}` and build:

```go
es := &elasticsearchcrd.Elasticsearch{
    Spec: elasticsearchcrd.ElasticsearchSpec{
        Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
            Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
                EndpointIngressSpec: shared.EndpointIngressSpec{
                    Enabled:    true,
                    Host:       "my-es.example.com",
                    TlsEnabled: tt.ingressTLS,
                    SecretRef:  tt.secretRef,
                },
            },
            Route: &elasticsearchcrd.ElasticsearchRouteSpec{
                EndpointRouteSpec: shared.EndpointRouteSpec{
                    Enabled:    true,
                    Host:       "my-es-route.example.com",
                    TlsEnabled: tt.routeTLS,
                    SecretRef:  tt.secretRef,
                },
            },
        },
        Tls: shared.TlsSpec{Enabled: tt.backendTLS},
    },
}
h := &ElasticsearchReconciler{}
got, err := h.computeElasticsearchUrl(context.Background(), es)
assert.NoError(t, err)
assert.Equal(t, tt.want, got)
```

Cases (mirror §5; at minimum cover rows 1, 4, 10, 11, 12, 13, 16, 18, 19 plus a SecretRef-set
variant). Example expected strings: `"https://my-es.example.com"`, `"http://my-es.example.com"`,
`"https://my-es-route.example.com"`, `"http://my-es-route.example.com"`. Use `ptr.To(true)`,
`ptr.To(false)`, and `nil` for the pointer fields, and `&v1.LocalObjectReference{Name: "cert"}` /
`nil` for `SecretRef`.

### 6.2 `internal/controller/kibana/compute_url_test.go` (new, `package kibana`)

Function `TestComputeKibanaUrl(t *testing.T)`.

Same shape as §6.1 but with `kibanacrd.Kibana` and `kibanacrd.KibanaSpec`. Note Kibana uses the
shared endpoint types directly:

```go
kb := &kibanacrd.Kibana{
    Spec: kibanacrd.KibanaSpec{
        Endpoint: shared.EndpointSpec{
            Ingress: &shared.EndpointIngressSpec{Enabled: true, Host: "my-kibana.example.com", TlsEnabled: tt.ingressTLS, SecretRef: tt.secretRef},
            Route:   &shared.EndpointRouteSpec{Enabled: true, Host: "my-kibana-route.example.com", TlsEnabled: tt.routeTLS, SecretRef: tt.secretRef},
        },
        Tls: shared.TlsSpec{Enabled: tt.backendTLS},
    },
}
h := &KibanaReconciler{}
got, err := h.computeKibanaUrl(context.Background(), kb, nil) // nil ConfigMap => basePath ""
assert.NoError(t, err)
assert.Equal(t, tt.want, got)
```

Expected strings: `"https://my-kibana.example.com"`, `"http://my-kibana.example.com"`,
`"https://my-kibana-route.example.com"`, `"http://my-kibana-route.example.com"`.

### 6.3 `internal/controller/cerebro/compute_url_test.go` (new, `package cerebro`)

Function `TestComputeCerebroUrl(t *testing.T)`.

Same shape, using `cerebrocrd.Cerebro` with `Endpoint: shared.EndpointSpec{...}` and no `Tls` field.
Cover rows 20, 21, 24, 25, 26, 28. Expected strings:
`"https://my-cerebro.example.com"`, `"http://my-cerebro.example.com"`,
`"https://my-cerebro-route.example.com"`, `"http://my-cerebro-route.example.com"`.

### Coexistence note

- The new files use `func TestComputeXxxUrl(t *testing.T)` (plain) while the existing envtest suites
  use `func TestXxxControllerSuite(t *testing.T)` + `suite.Run`. Both compile into the same package
  without conflict.
- Do not modify `suite_test.go` or the existing `*_controller_test.go` `assert.NotEmpty(...Status.Url)`
  assertions; they remain valid (the scheme change still yields a non-empty URL).
- Workstream B needs no new unit test: `api/shared/tls_test.go` already asserts the `true`-on-nil
  default, and Workstream B is a comment/schema-only change.

## 7. Error handling

No new error paths are introduced. The change only swaps the boolean condition inside branches that
already return `(string, nil)`. The load-balancer branch (which returns an error when the Service
cannot be fetched) and the internal-service branch are untouched, so their error behavior and output
are unchanged. Workstream B changes no executable code.

## 8. Validation commands

Run from the repo root, in order:

```bash
gofmt -l .                                  # expect no output
go vet ./...                                # expect clean
go build ./...                              # expect success
```

Targeted unit tests (does not require KUBEBUILDER assets; runs the new tests plus any other
non-suite tests in the package):

```bash
go test ./internal/controller/elasticsearch/ -run 'TestComputeElasticsearchUrl' -count 1 -v
go test ./internal/controller/kibana/        -run 'TestComputeKibanaUrl'        -count 1 -v
go test ./internal/controller/cerebro/       -run 'TestComputeCerebroUrl'       -count 1 -v
```

Full controller test suites (envtest-based; requires `KUBEBUILDER_ASSETS` per the Makefile
`test` target, i.e. `make test` or the equivalent `ES_OPERATOR_ENVTEST=true ... go test -p 1 ...`):

```bash
go test ./internal/controller/elasticsearch/... ./internal/controller/kibana/... ./internal/controller/cerebro/... -count 1
```

Lint: the repo has **no** `.golangci.yml`; the canonical lint is `make vet` (`go vet ./...`). The
project also documents `dagger call -m golang --src . lint` (see `CONTRIBUTE.md` line 97), but that
requires the Dagger CLI; `go vet ./...` is the mandatory, dependency-free check.

**Diff review (mandatory for Workstream B):**

```bash
git diff --stat
git diff -- api/shared/tls.go config/crd/bases/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml config/crd/bases/kibana.k8s.webcenter.fr_kibanas.yaml bundle/manifests/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml bundle/manifests/kibana.k8s.webcenter.fr_kibanas.yaml config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml bundle/manifests/elasticsearch-operator.clusterserviceversion.yaml
```

Confirm the diff touches **only**: (a) the three controller `.go` files + three new `*_test.go` files
from Workstream A; (b) `api/shared/tls.go` comment + the `tls.enabled` description text in the six
generated files from Workstream B. If `make bundle` produced unrelated changes (version bump, dropped
`v1alpha1` entries, or other descriptors), revert and apply §3.3 manually.

## 9. Branch strategy

- Default branch is `main` (see `CONTRIBUTE.md`: "use the `main` branch to start").
- Create feature branch `feat/ingress-tls-enabled` from `main`.
- Do **not** push or open a PR as part of coding; committing/pushing/PR is out of scope for this task.

## 10. Out of scope & risks

**Out of scope:**
- Route `tlsEnabled` CRD default: the reference `EndpointRouteSpec.TlsEnabled` carries
  `+kubebuilder:default=true` while the target does not (target `api/shared/endpoint.go:96` vs
  reference `api/shared/endpoint.go:97`). Runtime behavior is identical (`IsTlsEnabled()` returns
  `true` on nil), and the user's request does not cover this field; it remains out of scope.
- Any change to the ingress/route **builders** (they are already correct).
- Bundle version bump / catalog regeneration / publishing, and any broad regeneration beyond the
  `tls.enabled` description fix.
- PR creation / pushing / CI.

**Risks:**
- **Behavior change for users relying on the buggy scheme.** Any automation that asserted
  `status.url == http://...` for a TLS-enabled ingress (i.e. `SecretRef` unset with `tlsEnabled`
  omitted/true) will now see `https://...`. This is the intended correction and matches the reference
  operator, but it is a user-visible change.
- `SecretRef`-set-but-TLS-disabled (`tlsEnabled: false` + `Spec.Tls.Enabled: false`) now reports
  `http://` even though a `SecretRef` is present — this is correct, because the builder emits no TLS
  block in that case (rows 11 and 25).
- **`make bundle` regeneration may over-rewrite.** The base CSV still contains stale `v1alpha1`
  owned-CRD entries (the api tree is `v1`-only), so a full regen could drop/rewrite them and produce
  unrelated diffs. Mitigate per §3.2/§3.3: prefer `make manifests` (targeted) and treat `make bundle`
  output as verify-then-accept, falling back to the manual edits if the diff grows beyond the
  `tls.enabled` description.

## 11. Step-by-step implementation checklist

### Phase A — status URL scheme

1. Create branch `feat/ingress-tls-enabled` from `main`.
2. Edit `internal/controller/elasticsearch/elasticsearch_controller.go` — replace the ingress
   condition (line 375) with `es.Spec.Endpoint.Ingress.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled()`
   and the route condition (line 383) with
   `es.Spec.Endpoint.Route.IsTlsEnabled() || es.Spec.Tls.IsTlsEnabled()`.
3. Edit `internal/controller/kibana/kibana_controller.go` — replace the ingress condition (line 316)
   with `kb.Spec.Endpoint.Ingress.IsTlsEnabled() || kb.Spec.Tls.IsTlsEnabled()` and the route
   condition (line 324) with `kb.Spec.Endpoint.Route.IsTlsEnabled() || kb.Spec.Tls.IsTlsEnabled()`.
4. Edit `internal/controller/cerebro/cerebro_controller.go` — replace the ingress condition
   (line 285) with `cb.Spec.Endpoint.Ingress.IsTlsEnabled()` and the route condition (line 293) with
   `cb.Spec.Endpoint.Route.IsTlsEnabled()`.
5. Add `internal/controller/elasticsearch/compute_url_test.go` with `TestComputeElasticsearchUrl`
   (table-driven, `k8s.io/utils/ptr`, `assert.Equal`).
6. Add `internal/controller/kibana/compute_url_test.go` with `TestComputeKibanaUrl`.
7. Add `internal/controller/cerebro/compute_url_test.go` with `TestComputeCerebroUrl`.

### Phase B — `TlsSpec.Enabled` default true

8. Edit `api/shared/tls.go` line 8: change `// Default to false` → `// Default to true`.
9. Regenerate CRD bases: run `make manifests` and confirm only the two base-CRD `tls.enabled`
   descriptions changed. (Skip if tooling unavailable; use the §3.3 manual table instead.)
10. Regenerate bundle + CSV: run `make bundle`, then inspect `git diff`. Accept only if the diff is
    limited to the `tls.enabled` description text; otherwise revert the generated files and apply the
    §3.3 manual table.
11. If not regenerating (or reverting), apply the §3.3 manual edits to rows 2–10 of the fallback
    table (and confirm row 1 / line 469 already-correct cases are untouched).

### Phase C — validation

12. Run `gofmt -l .` (expect no output), `go vet ./...`, `go build ./...`.
13. Run the targeted `go test -run 'TestCompute...'` commands from §8.
14. Run the full envtest suites for the three packages per §8 (or `make test`).
15. Run the §8 `git diff` review and confirm only the expected files/lines changed (no builders, no
    unrelated CRD/CSV changes).
