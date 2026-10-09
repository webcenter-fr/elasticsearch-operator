# OpenShift (OCP) Readiness Backport

Backport the OpenShift deployment behavior of the reference project
`/projects/opensearch-operator-k8s` into `/projects/elasticsearch-operator`,
closing the remaining OCP-readiness gaps while preserving Kubernetes-only
behavior.

---

## 1. Goal & scope

### Goal
Make the operator fully OpenShift-ready by closing the gaps between the
reference (`opensearch-operator-k8s`) and the target (`elasticsearch-operator`)
for: route support, SCC RoleBindings, ServiceAccount wiring, workload
NetworkPolicies, and — the one real functional gap — the **webhook bootstrap
NetworkPolicy** plus the missing **`GetOperatorNamespace()`** helper.

### In scope
- Add `pkg/helper.GetOperatorNamespace()` (reads `POD_NAMESPACE`).
- Call `controller.EnsureNetworkPolicyForWebhook(...)` in `cmd/main.go` after
  webhook setup (operator-sdk-extra **v3** context-first signature).
- Remove the stale/broken static manifest `config/network-policy/`.
- Add a unit test for the new helper.
- Document parity items (no code change) so a future maintainer can re-audit.

### Explicitly out of scope
- **Strimzi** capability flag — the target must NOT gain `HasStrimzi`.
- **ClusterRoleBinding to SCC** — NOT implemented. The canonical/OpenShift
  approach (and what the reference does) is a **namespaced RoleBinding** to the
  SCC ClusterRole. A ClusterRoleBinding would over-grant `use` across all
  namespaces and is unnecessary. See §3.3 for the evidence.
- Cerebro CSV `{Route,route.openshift.io/v1}` annotation omission — cosmetic,
  also present in the reference; do NOT change (keeps diff minimal).
- Any change to Route builders, SCC builders, ServiceAccount builders, workload
  NetworkPolicy builders, RBAC markers, or envtest suites — already at parity.

---

## 2. Gap analysis (exhaustive)

Legend: **parity** (no change) / **partial** / **missing**.

### 2.1 Route support

| Reference path | Target path | Status |
|---|---|---|
| `internal/controller/opensearch/route_builder.go` | `internal/controller/elasticsearch/route_builder.go` | parity |
| `internal/controller/dashboard/route_builder.go` | `internal/controller/kibana/route_builder.go` | parity |
| `internal/controller/logstash/route_builder.go` | `internal/controller/logstash/route_builder.go` | parity |
| `internal/controller/filebeat/route_builder.go` | `internal/controller/filebeat/route_builder.go` | parity |
| `internal/controller/cerebro/route_builder.go` | `internal/controller/cerebro/route_builder.go` | parity |
| `internal/controller/*/route_reconciler.go` (5) | `internal/controller/*/route_reconciler.go` (5) | parity |
| controller wiring `if kubeCapability.HasRoute { append route reconciler }` + `Owns(&routev1.Route{})` | same in all 5 controllers | parity |

Verified identical per component: TLS `edge`/`reencrypt`, `ExternalCertificate`,
`DestinationCACertificate`, `InsecureEdgeTerminationPolicyRedirect`,
`k8sbuilder.MergeK8s` of user `RouteSpec`, host/path (`/`), labels/annotations,
target service + `TargetPort http`. Logstash/filebeat use `*i.Spec.DeepCopy()`
user-provided spec in both repos.

- **metricbeat has no Route** in the target — correct and intentional
  (metricbeat exposes no HTTP endpoint; it has no Service/Ingress/Route, only
  ConfigMap/StatefulSet/PDB/Secret/SA/RoleBinding). The reference has no
  metricbeat component at all. No action.

### 2.2 SCC RoleBinding / ServiceAccount / serviceAccountName

| Item | Reference | Target | Status |
|---|---|---|---|
| RoleBinding builder `system:openshift:scc:anyuid` + `privileged` on `IsSetVMMaxMapCount()` (opensearch) | yes | `elasticsearch/rolebinding_builder.go` | parity |
| RoleBinding builder `anyuid` only (dashboard/kibana, logstash, filebeat) | yes | kibana/logstash/filebeat/metricbeat | parity |
| ServiceAccount builder returns `nil` when not OpenShift | yes | yes (all components) | parity |
| `serviceAccountName` set only on OpenShift in Deployment/StatefulSet builders | yes | yes | parity |
| RBAC markers `security.openshift.io` `securitycontextconstraints` `use` | yes | yes (`config/rbac/role.yaml`) | parity |
| RoleBinding/SA list/watch/ownership + tests per component | yes | yes | parity |

**cerebro**: no ServiceAccount/RoleBinding in either repo (runs non-root,
no SCC binding). `newDeploymentReconciler(c, recorder, kubeCapability.HasRoute)`
present in both. Parity.

**metricbeat** (target-only component): fully wired for SA + RoleBinding
(`system:openshift:scc:anyuid`), no Route. Correct.

### 2.3 RoleBinding vs ClusterRoleBinding — contradiction resolved

The user request mentioned "add clusterRoleBinding to scc (any-scc)". **The
reference does not use ClusterRoleBinding.** Both repos bind the workload
ServiceAccount to the SCC ClusterRole via a **namespaced RoleBinding**:

```go
RoleRef: rbacv1.RoleRef{ Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io",
    Name: "system:openshift:scc:anyuid" }
```

This is the canonical OpenShift pattern (grants `use` of the SCC only to the
named SA in one namespace). **No change required.** A ClusterRoleBinding is NOT
part of the backport.

### 2.4 Workload NetworkPolicy

`internal/controller/{elasticsearch,kibana,logstash}/networkpolicy_builder.go`
(+ reconcilers) exist in the target with the operator-namespace pod-selector
label `elasticsearch-operator.k8s.webcenter.fr: "true"` (matching the manager
pod labels in `config/manager/manager.yaml`). Parity with reference. No change.

### 2.5 Webhook NetworkPolicy bootstrap — **MISSING**

- Reference `cmd/main.go` (~lines 280-304) calls
  `controller.EnsureNetworkPolicyForWebhook(...)` after webhook setup.
- Target `cmd/main.go` (ends at line 409) **never calls it**.
- Target also carries a **stale static manifest**
  `config/network-policy/allow-webhook-traffic.yaml` that is:
  - not referenced by any kustomization (dead),
  - wrong port (**443** vs the actual pod port **9443**),
  - restricted to `webhook: enabled` namespaces (breaks OCP kube-apiserver
    admission traffic).
  - The reference has **no** `config/network-policy/` directory (uses the
    code-based helper only).

**Action**: add the code-based call in `main.go`; delete
`config/network-policy/`.

### 2.6 `GetOperatorNamespace()` — **MISSING**

- Reference `pkg/helper/namepsace.go` (sic) + `pkg/helper/namespace_test.go`.
- Target `pkg/helper/` has only `hash.go`, `yaml.go`, `config.go`,
  `statefulset.go` (+ tests). No namespace helper.
- operator-sdk-extra **v3** (`/projects/pkg/mod/.../operator-sdk-extra/v3@v3.0.9/pkg/helper`)
  does **not** provide it (verified via grep — only `EnsureNetworkPolicyForWebhook`
  in `pkg/controller`).

### 2.7 Scheme registration / capability detection / RBAC / kustomize / CSV

| Item | Reference | Target | Status |
|---|---|---|---|
| `routev1.AddToScheme` in `init()` | yes | yes (`cmd/main.go:77`) | parity |
| `helper.HasCRD(clientStd, routev1.SchemeGroupVersion)` → `HasRoute` | yes | yes (`cmd/main.go:217`) | parity |
| `common.KubernetesCapability{HasRoute,HasPrometheus}` (no Strimzi) | yes | yes | parity |
| RBAC route markers `route.openshift.io` `routes` + `routes/custom-host` | yes | yes (generated role.yaml) | parity |
| `config/crd/externals/crd-route.yaml` | yes | yes | parity |
| manager env `POD_NAMESPACE` (downward API) | yes | yes (`config/manager/manager.yaml`) | parity |
| CSV/CRD Route annotations (es/kibana/logstash/filebeat) | yes | yes | parity |
| CSV Route annotation (cerebro) | **omitted** | **omitted** | parity (both omit; leave) |
| OpenShift kustomize overlay | none | none | parity |

### 2.8 Tests / testdata

- All 8 target `internal/controller/*/suite_test.go` already use envtest with
  `CRDDirectoryPaths = [config/crd/bases, config/crd/externals]` and compute
  `kubeCapability.HasRoute` via `helper.HasCRD(...)`. Parity with reference.
- Golden YAML testdata for routes/rolebindings/serviceaccounts/networkpolicies
  exists per component (compared with `test.EqualFromYamlFile`). Parity.
- **Missing**: no unit test for the new `GetOperatorNamespace()` helper (to be
  added with the helper).

---

## 3. Exact changes (files, signatures, snippets)

### 3.1 `pkg/helper/namespace.go` — **NEW**

```go
package helper

import (
	"os"

	"emperror.dev/errors"
)

const (
	operatorNamespaceEnvVar = "POD_NAMESPACE"
)

func GetOperatorNamespace() (ns string, err error) {
	ns, found := os.LookupEnv(operatorNamespaceEnvVar)
	if !found {
		return "", errors.Errorf("%s must be set", operatorNamespaceEnvVar)
	}

	return ns, nil
}
```

Notes:
- Uses `emperror.dev/errors` (project's direct dependency; `errors.Errorf` is
  already used across the repo). Do NOT import `github.com/pkg/errors`.
- Mirrors reference `pkg/helper/namepsace.go` exactly (same env var, same
  message `"POD_NAMESPACE must be set"`).

### 3.2 `pkg/helper/namespace_test.go` — **NEW**

```go
package helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOperatorNamespace(t *testing.T) {
	// When env not exist
	_, err := GetOperatorNamespace()
	assert.Error(t, err)

	// When env exist
	_ = os.Setenv(operatorNamespaceEnvVar, "test")
	ns, err := GetOperatorNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "test", ns)
}
```

### 3.3 `cmd/main.go` — **MODIFY**

**Imports to add** (keep existing imports; note existing `"context"` and
`"sigs.k8s.io/controller-runtime/pkg/client"` — add the `client` import and the
`localhelper` alias):

```go
	"sigs.k8s.io/controller-runtime/pkg/client"

	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
```

**Insert after the webhook block** — i.e. after the closing brace of
`if os.Getenv("ENABLE_WEBHOOKS") != "false" { ... }` (current line 276) and
before `// Init controllers` (current line 278). Recommended: place the block
**inside** the `ENABLE_WEBHOOKS != "false"` guard so the NP is only created
when webhooks actually run, and so local `make run` (no `POD_NAMESPACE`, but
webhooks enabled) still requires the var only when relevant. Reference places it
unconditionally after the guard; the gated placement is a deliberate, safer
deviation. Either placement is functionally correct on OCP (where `POD_NAMESPACE`
is always injected).

```go
	// Add NetworkPolicy for webhook (OCP/apiserver must reach TCP 9443).
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create client for webhook networkPolicy")
		os.Exit(1)
	}
	namespace, err := localhelper.GetOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get operator namespace", "namespace", "POD_NAMESPACE")
		os.Exit(1)
	}
	if err = controller.EnsureNetworkPolicyForWebhook(
		context.Background(),
		cl,
		logrus.NewEntry(log),
		namespace,
		map[string]string{
			"app.kubernetes.io/name": "elasticsearch-operator",
		},
		map[string]string{
			"control-plane": "elasticsearch-operator",
		},
	); err != nil {
		setupLog.Error(err, "unable to create networkPolicy", "controller", "core")
		os.Exit(1)
	}
```

**Why a direct client (not `mgr.GetClient()`):** `mgr.GetClient()` is
cache-backed and will block on the initial `Get` before `mgr.Start()`. The
reference creates an uncached client with `client.New(cfg,
client.Options{Scheme: scheme})` for the same reason. Keep this.

**Label/selector rationale** (verified against kustomize `namePrefix`):
- `namePrefix: elasticsearch-operator-` in `config/default/kustomization.yaml`
  renames **resources**, not labels/selectors.
- Manager pod labels (`config/manager/manager.yaml`): `control-plane:
  elasticsearch-operator` (+ `elasticsearch-operator.k8s.webcenter.fr: "true"`,
  `kibana-operator.k8s.webcenter.fr: "true"`). Deployment
  `spec.selector.matchLabels` = `control-plane: elasticsearch-operator`.
- Therefore the NetworkPolicy `PodSelector.MatchLabels =
  {"control-plane": "elasticsearch-operator"}` selects the manager pod
  deterministically. The NP `Labels = {"app.kubernetes.io/name":
  "elasticsearch-operator"}` matches the existing webhook Service label and the
  reference's `app.kubernetes.io/name` convention.

**Helper behavior (v3.0.9, verified)** — `EnsureNetworkPolicyForWebhook`
creates/updates a NetworkPolicy named **`allow-webhook-access-from-any`** in the
given namespace with:
- `PodSelector.MatchLabels = podSelectors` (param),
- one Ingress rule: `From: []` (allow all sources), `Ports: [{Protocol: TCP,
  Port: 9443}]`,
- 3-way-diff via `patch.DefaultPatchMaker` + last-applied annotation on create.

### 3.4 `config/network-policy/allow-webhook-traffic.yaml` — **DELETE**

Delete the whole `config/network-policy/` directory. It is dead, uses the wrong
port, and its `webhook: enabled` namespace restriction would break OCP webhook
admission. The reference has no such directory.

---

## 4. Ordered implementation checklist

1. Create branch `feat/openshift-ocp-backport` from `main`.
2. Add `pkg/helper/namespace.go` (§3.1).
3. Add `pkg/helper/namespace_test.go` (§3.2).
4. Modify `cmd/main.go` (§3.3): add imports + bootstrap block.
5. Delete `config/network-policy/` (§3.4).
6. Run `make generate` and `make manifests` (confirm no diff beyond expected —
   this change adds no markers, so output should be unchanged).
7. Run `make build`.
8. Run `make test`.
9. Run `golangci-lint run --timeout 5m` and `govulncheck ./...` if available.
10. Commit with conventional commits, push, open PR via `gh pr create`.

---

## 5. Test plan

### Unit
- `pkg/helper/namespace_test.go` — `TestGetOperatorNamespace`: unset →
  error; set → returns value (§3.2).

### envtest / controller suites (existing, no change)
- `make test` runs all suites
  (`./api/... ./internal/controller/... ./pkg/...`) with
  `KUBEBUILDER_ASSETS` + `ES_OPERATOR_ENVTEST=true`.
- Existing suites already exercise: route CRD present/absent (via
  `helper.HasCRD`), RoleBinding/SA created only when OpenShift, workload
  NetworkPolicy, TLS edge/reencrypt golden YAML, ownership/delete.

### Edge cases to verify manually (documented, not new automated tests required)
- `POD_NAMESPACE` unset → `GetOperatorNamespace` returns error, `main.go` logs
  and exits non-zero.
- `POD_NAMESPACE` set → NP `allow-webhook-access-from-any` created in that
  namespace, TCP 9443, selecting `control-plane: elasticsearch-operator`.
- NP already exists with drift → helper updates via 3-way diff (no error).
- `vm.max_map_count` set/unset → `system:openshift:scc:privileged` /
  `...:anyuid` (already covered by `elasticsearch/rolebinding_builder_test.go` +
  `testdata/rolebinding_anyuid.yml` / `rolebinding_default.yml`).
- No routes configured → `buildRoutes` returns `nil`, no Route created.
- Route CRD absent (plain K8s) → `HasRoute=false`, no route reconciler, no
  RoleBinding/SA. No regression.

---

## 6. Error handling & validation strategy per change

- **`GetOperatorNamespace`**: returns `emperror.dev/errors.Errorf("POD_NAMESPACE
  must be set")` when unset; caller (`main.go`) logs via `setupLog.Error` and
  `os.Exit(1)` (matches existing `main.go` error style; the reference uses
  `fmt.Println`, which is not the target's convention).
- **`EnsureNetworkPolicyForWebhook`**: returns wrapped errors on
  Get/Create/Update/annotate failures; caller logs and exits. On OCP the var is
  always injected (downward API in `manager.yaml`), so this path is safe.
- **Deletion of static manifest**: no runtime impact (was never applied).

---

## 7. Local validation commands (exact)

From `Makefile` and `ci/dagger/main.go` (no `.golangci.yml` in repo):

```bash
# build (generate -> fmt -> vet -> go build)
make build

# full unit + envtest suite (manifests -> generate -> fmt -> envtest -> go test)
make test

# controller-gen idempotency (rbac + crd + webhook, then strip CEL validations)
make manifests

# deepcopy generation
make generate

# vet / fmt (already part of build, but runnable alone)
make vet
make fmt
```

CI-equivalent (Dagger pipeline, from `ci/dagger` + `dagger-library-go/golang`):

```bash
golangci-lint run --timeout 5m
govulncheck ./...
```

Expected outcomes: all commands exit 0; `make manifests`/`make generate`
produce no diff (no new markers added); `make test` green.

### Branch & PR workflow
- Branch: `feat/openshift-ocp-backport` created from `main`.
- Conventional commits (e.g. `feat(openshift): add webhook NetworkPolicy
  bootstrap and POD_NAMESPACE helper`).
- `gh pr create` at the end. **If `gh` or the git remote is unavailable, the
  coder MUST report that instead of failing silently.**

---

## 8. Risks / rollout / backward-compat

- **Kubernetes-only regression risk: none.** The change only (a) adds a
  namespaced NetworkPolicy in the operator's own namespace (port 9443) when
  webhooks run, and (b) adds a helper. No route/SA/RoleBinding behavior changes.
- **`make run` behavior**: with the recommended gated placement, the NP is only
  created when `ENABLE_WEBHOOKS != "false"`. If webhooks are enabled and
  `POD_NAMESPACE` is unset (local run), the manager will exit with a clear
  message. This matches reference behavior; document that local runs should set
  `POD_NAMESPACE` (or `ENABLE_WEBHOOKS=false`).
- **NP allow-all ingress**: the generated NP allows ingress to TCP 9443 from all
  sources (apiserver reachability). This is intentional per the v3 helper
  security note; on hardened clusters it can be tightened later (out of scope).
- **No new third-party deps**: uses existing `emperror.dev/errors`,
  `controller-runtime`, `k8s.io/api/networking/v1` (transitively via the v3
  helper).
