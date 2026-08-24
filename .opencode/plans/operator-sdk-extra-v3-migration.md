# Migrate operator-sdk-extra v2 → v3 (SSA + cert lifecycle)

## 0. Executive summary

Migrate `github.com/disaster37/operator-sdk-extra/v2` (v2.0.10) → `github.com/disaster37/operator-sdk-extra/v3` **v3.0.7** (the operator's current pin; the cert redesign + generic subject/key customization landed in v3.0.6 = commit `8f135ed0…`, module path `/v3`, go 1.26). v3.0.7 is consumed directly via `go get …@v3.0.7` (§3). (Historical note: the `v3.0.3` tag is a mis-tag of the v2 commit `8c9ead49` — module `/v2`, go 1.24 — and must not be used.) v3 makes two structural changes that force a real migration, not just a rename:

1. **Multiphase + sentinel reconcilers use Server-Side Apply (SSA)** instead of client-side 3-way strategic-merge diff (`k8s-objectmatcher`). The multiphase step reconciler drops the `dryRun bool` ctor param and `Create()`/`Update()` methods in favor of a `fieldManager string` ctor param and a single `Apply()`; `Diff()` no longer takes `ignoreDiff ...patch.CalculateOption`.
2. **v3 ships a certificate/CA lifecycle** (`pkg/controller/certificate` with `byo`/`selfmanaged`/`certmanager` backends) plus a **workflow saga** (`pkg/apis/workflow` + `pkg/controller/workflow`).

**Decision (confirmed by user): full v3 cert adoption** — replace the custom condition-based TLS FSM (the `secret_tls_reconciler.go` family) with the library's reusable `rotation.NewTLSStep` saga, and rework the Elasticsearch transport TLS into a **shared CA + per-node leaf** model using the library's `selfmanaged/pernode` backend. The CRD **spec API stays unchanged** except for two additive optional fields `caRenewalDays` and `KeyComplexity` (RSA→ECDSA opt-in); all cert attributes (O/OU/SAN/CN/IP) keep being computed by the controller from cluster topology via a `TLSSpecProvider` + `NodeSpecProvider` + `rotation.WithCertificateCustomizer`. **Trust-model verified:** ES transport trust is CA-anchored (`verification_mode=full` + shared `ca.crt`), so adding a node without rollout cannot break trust (§4.4.5). The library (v3.0.7) already provides IP SANs, per-node certs, CA/leaf validity + renewal, the rotation saga, the generic `CertificateSubject` (O/OU/Country/Locality/Province), and `KeyAlgorithm`/`KeySize` (ECDSA/RSA); the one remaining gap (`CARenewalDays`, a CA-specific renewal window) is handled by the dedicated library plan `operator-sdk-extra-certificate-customization.md`. Preferred path (option a) consumes the library; an interim local `TLSBackend` (option b, retaining `goca`) is the fallback. See §4.4 and R1.

The **remote** reconciler pattern (Elasticsearch/Kibana API objects) is largely unchanged (still 3-way merge via `generic-objectmatcher` + `status.lastAppliedConfiguration`) — only import paths change.

---

## 1. Current-state inventory (verified from repo)

### 1.1 Module / toolchain
- `go.mod`: module `github.com/webcenter-fr/elasticsearch-operator`, `go 1.26.0`.
- Key deps: `operator-sdk-extra/v2 v2.0.10`, `controller-runtime v0.24.1`, `k8s.io/* v0.36.3`, `k8s-objectmatcher v1.8.8`, `generic-objectmatcher v1.0.2`, `goca v1.0.5`, `k8sbuilder v1.0.3`, `mergo v1.0.2`, `emperror.dev/errors v0.8.1`, `es-handler/v8 v8.1.5` (replaced by `./third_party/es-handler`), `go-kibana-rest/v8`, `kb-handler/v8`.
- Kubebuilder scaffold: `PROJECT` layout `go.kubebuilder.io/v4`, `multigroup: true`, 23 resources. `Makefile` has `manifests`, `generate`, `fmt`, `vet`, `test` (envtest, `KUBEBUILDER_ASSETS`, `TEST=true`), `build`, etc. `ENVTEST_K8S_VERSION = 1.25.x`, `CONTROLLER_TOOLS_VERSION = v0.21.0`.
- `bin/setup-envtest`, `config/crd/{bases,externals}`, `config/webhook` are used by envtest suites.

### 1.2 operator-sdk-extra/v2 subpackages in use
- `pkg/controller` — `Controller`, `NewController`, `ReadyCondition`, `RunningPhase`, `StartingPhase`, `SetupIndexerWithManager`, `SetupWebhookWithManager` (also used by every `suite_test.go`).
- `pkg/controller/multiphase` — multiphase reconciler stack (6 controllers: elasticsearch, kibana, logstash, filebeat, metricbeat, cerebro).
- `pkg/controller/remote` — remote reconciler stack (elasticsearchapi: 10 kinds; kibanaapi: 3 kinds).
- `pkg/helper` — `Get`, `ToSlicePtr`, `ToSliceOfObject`, `RandomString`, `DeleteItemFromSlice`, `DiffMapString`, `GetZapLogLevelFromEnv`, `GetZapFormatterFromDev`, `GetLogrusLogLevelFromEnv`, `GetLogrusFormatterFromEnv`, `GetWatchNamespaceFromEnv`, `StringToSlice`, `PrintVersion`, `GetKubeClientTimeoutFromEnv`, `HasCRD`.
- `pkg/apis`, `pkg/apis/shared` (`PhaseName`, `ConditionName`, `FinalizerName`), `pkg/apis/multiphase` (`DefaultMultiPhaseObjectStatus`), `pkg/apis/remote` (`DefaultRemoteObjectStatus`), `pkg/apis.MapAny`.
- `pkg/object` — `MultiPhaseObjectStatus`, `RemoteObjectStatus` (in `api/*/v1/*_func.go`).
- `pkg/test` — envtest helpers (`test.RunWithTimeout` in `suite_test.go`).
- `pkg/mock` — mocks (elasticsearchapi/kibanaapi `suite_test.go`).

### 1.3 Controllers and step reconcilers (multiphase)
All multiphase controllers share the same shape: a `*_controller.go` that embeds `controller.Controller` + `multiphase.MultiPhaseReconciler` + `multiphase.MultiPhaseReconcilerAction`, holds `stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*X, client.Object]` built with `multiphase.NewObjectMultiPhaseStepReconcilerAction[*X, *Y, client.Object](newYReconciler(...))`, and each step reconciler is a `*_reconciler.go` embedding `multiphase.MultiPhaseStepReconcilerAction[*X, *Y]` built with `multiphase.NewMultiPhaseStepReconcilerAction[*X, *Y](client, phase, condition, recorder)`.

- `internal/controller/elasticsearch/` — steps: serviceaccount, rolebinding, secret_tls, secret_credential, license, configmap, service, pdb, networkpolicy, statefulset, system_user, ingress, loadbalancer, metricbeat, deployment_exporter (+ podmonitor if Prometheus, + route if OpenShift). Files: `elasticsearch_controller.go`, `elasticsearch_watcher.go`, `helper.go`, `*_builder.go` (15+), `*_reconciler.go` (15+), `secret_tls_builder.go`, `suite_test.go`, `elasticsearch_controller_test.go`, `*_builder_test.go`, `testdata/`.
- `internal/controller/kibana/`, `logstash/`, `filebeat/`, `metricbeat/` — same pattern (their own `secret_tls_reconciler.go`, `statefulset_reconciler.go`, `secret_ca_elasticsearch_reconciler.go`, etc.).
- `internal/controller/cerebro/` — steps: secret_application, configmap, service, deployment, ingress, loadbalancer (+ route).
- `internal/controller/common/` — `controller.go` defines a **local** `DefaultControllerRateLimiter()` (keep; not an SDK dependency).

### 1.4 Remote controllers (unchanged pattern)
- `internal/controller/elasticsearchapi/` — user, license, role, rolemapping, indexlifecyclepolicy, snapshotlifecyclepolicy, snapshotrepository, componenttemplate, indextemplate, watch. Each has `*_controller.go` (embeds `controller.Controller` + `remote.RemoteReconciler`), `*_reconciler.go` (embeds `remote.RemoteReconcilerAction`, overrides `GetRemoteHandler`/`Read`/`Diff`/`Delete`/`OnSuccess`), `*_api_client.go` (embeds `remote.RemoteExternalReconciler`, implements `Build/Get/Create/Update/Delete/Diff`).
- `internal/controller/kibanaapi/` — userspace, role, logstashpipeline.
- **Elasticsearch client stack (to migrate, §4.5.1):** `internal/controller/elasticsearchapi/*` and `internal/controller/elasticsearch/*` use `github.com/disaster37/es-handler/v8` (`ElasticsearchHandler` interface + `NewElasticsearchHandler`) + the official `github.com/elastic/go-elasticsearch/v8` (only to build `elastic.Config`) + `github.com/olivere/elastic/v7` **types** (`XPackSecurityPutUserRequest`, `XPackSecurityRoleMapping`, `XPackInfoLicense`, `IndicesGetIndexTemplate`, `IndicesGetComponentTemplate`, `XPackIlmGetLifecycleResponse`, `SnapshotRepositoryMetaData`, `XPackWatch`, `ClusterHealthResponse`). The `third_party/es-handler` fork is gone (consumed from the proxy).
- Remote `Diff()` signatures currently take `ignoreDiff ...patch.CalculateOption` where `patch` = `github.com/disaster37/generic-objectmatcher/patch` (NOT k8s-objectmatcher). The ES/Kibana API client `Diff()` delegates to `es-handler`/`kb-handler` which use `generic-objectmatcher`.

### 1.5 3-way diff / diffIgnore inventory (the things SSA removes)
- `k8s-objectmatcher/patch` (K8s client-side diff — **must go**):
  - `patch.DefaultPatchMaker.Calculate(currentSts, expectedSts, patch.CleanMetadata(), patch.IgnoreStatusFields(), patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus())` in `statefulset_reconciler.go` ×4 (elasticsearch:310, metricbeat:238, filebeat:315, logstash:273).
  - `patch.DefaultAnnotator.SetLastAppliedAnnotation(...)` in `secret_tls_reconciler.go` ×4 (elasticsearch, logstash, kibana, filebeat) and `patch.LastAppliedConfig` in `pkg/helper/diff.go`.
  - Imports of `k8s-objectmatcher/patch` in `pdb_reconciler.go` (elasticsearch/logstash/kibana/filebeat) and in `*_controller_test.go` files (elasticsearch, kibana, logstash, filebeat, metricbeat, cerebro) for test assertions.
- Local diff helpers `pkg/helper/diff.go` → `DiffLabels`, `DiffAnnotations`, `DiffOwnerReferences` (each delegates to `helper.DiffMapString`). Used only by `secret_tls_reconciler.go` ×4 and `secret_tls_builder_test.go`. **Must go** (see §4.6).
- `generic-objectmatcher/patch` (remote API 3-way merge — **keep**, it is a different mechanism than SSA): used in every `*_api_client.go` `Diff()` and inside `third_party/es-handler/*`. Unaffected by this migration.

### 1.6 Existing certificate management (to be reworked, NOT rewritten from scratch)
- `internal/controller/elasticsearch/secret_tls_reconciler.go` (+ builder `secret_tls_builder.go`): condition-based FSM with phases `TlsPhaseCreate/UpdatePki/PropagatePki/UpdateCertificates/PropagateCertificates/CleanTransportCA/Normal/Reconcile` and conditions `TlsGeneratePki/PropagatePki/GenerateCertificate/PropagateCertificate/Ready/Blackout`. Generates transport **PKI (CA)** + a **per-node certificate** per pod (all stored as `<node>.crt`/`<node>.key` in ONE shared Secret `GetSecretNameForTlsTransport` = `<name>-tls-transport-es`, plus `ca.crt`) + API PKI + API cert, using `goca`.
- **How O/SAN/CN are computed today (must be preserved):** in `secret_tls_builder.go` (`buildTransportPkiSecret/buildTransportSecret/buildApiPkiSecret/buildApiSecret/generateNodeCertificate/generateApiCertificate`) — transport CA: `CN=<name>-transport`, `O=<name>`, `OU=transport`, `Country/Locality/Province=internal`, `Valid=ValidityDays||397`, `KeyBitSize=KeySize||2048`; transport node cert: `CN=<nodeName>`, `O=<name>`, `OU=<nodeGroupName>`, `DNSNames=` node headless/global service DNS names + `nodeName` + `.ns`/`.ns.svc` variants, `IPAddresses=[127.0.0.1]`; API CA: `CN=<name>-api`, `O=<name>`, `OU=api`; API cert: `CN=<name>`, `O=<name>`, `OU=api`, `DNSNames=` global service + node-group service names + `AltNames`, `IPAddresses=` `AltIps`. Logstash builder mirrors this (CA `CN=<name>-logstash`/`OU=logstash`; per-`Pki.Tls` certs with service/ingress SANs + `AltNames`/`AltIps`).
- **Mount mechanism (StatefulSet):** `statefulset_builder.go` mounts ONE `node-tls` Secret volume (`SecretName=GetSecretNameForTlsTransport(es)`) and the `init-filesystem` container copies `${POD_NAME}.crt`/`${POD_NAME}.key`/`ca.crt` from it into `/mnt/config/transport-cert/`; `configmap_builder.go` wires `xpack.security.transport.ssl.certificate = .../transport-cert/${POD_NAME}.crt`. Each pod therefore selects its OWN cert from the shared Secret by pod name — there is no per-pod Secret volume.
- **Zero-rollout membership today:** `statefulset_builder.go` uses `s.Annotations[".../sequence"]` as the pod-template checksum if present, else hashes `s.Data`. On node add/remove, `secret_tls_reconciler.go` edits the shared Secret's per-node keys WITHOUT bumping `sequence` (comment: "Keep existing sequence to not rolling restart all nodes"), so membership changes do NOT trigger a StatefulSet rollout; the new pod reads its cert at init.
- `internal/controller/{logstash,kibana,filebeat}/secret_tls_reconciler.go` + builders: simpler single-cert TLS (CA + leaf), regenerate-on-expiry (no CA-rotation saga, no blackout/bootstrapping).
- `pkg/pki/common.go`: `NeedRenewCertificate`, `LoadRootCA` (goca-based). `DefaultCertificateValidity = 397`, `DefaultRenewCertificate` commented out, `KeyBitSize = 2048`.
- Spec sources: `api/shared/tls.go` `TlsSpec` (`Enabled`, `SelfSignedCertificate{AltIps,AltNames}`, `CertificateSecretRef`, `ValidityDays`, `RenewalDays`, `KeySize`) used by **elasticsearch** (`Spec.Tls`) and **kibana** (`Spec.Tls`); `api/logstash/v1/logstash_types.go` `Pki` spec (own `ValidityDays/RenewalDays/KeySize` + `Tls map[string]LogstashTlsSpec`); `api/beat/v1/filebeat_types.go` `FilebeatPkiSpec` (`Pki` with `Tls map[string]shared.TlsSelfSignedCertificateSpec`). **No CRD has a CA-specific renewal/validity field today.**

---

## 2. Target v3 API surface (verified against v3.0.6 = commit `8f135ed0…`, the cert-redesign commit; operator pins v3.0.7; `v3.0.3` is a mis-tag of the v2 commit)

Sources: local checkout `/projects/operator-sdk-extra` (branch `v3`), pkg.go.dev `operator-sdk-extra/v3`, and upstream `documentations/migration-v2-to-v3.md`, `multi-phase-reconciler.md`, `remote-reconciler.md`, `tls-and-workflow.md`.

### 2.1 Module path + renames
- Module: `github.com/disaster37/operator-sdk-extra/v3`, pin **v3.0.7** (cert API == v3.0.6, commit `8f135ed0…`). The cert redesign (§2.4) + generic subject/key customization are present. **`v3.0.3` is a mis-tag** (commit `8c9ead49`, module `/v2`, go 1.24) — do not use it. Every import `/v2/` → `/v3/`. Go ≥ 1.26 (repo already 1.26.0).
- `controller` package constants/setup helpers are unchanged in signature: `NewController()`, `ReadyCondition`/`RunningPhase`/`StartingPhase`, `SetupIndexerWithManager(mgr, ...Indexer)`, `SetupWebhookWithManager(mgr, client, ...WebhookRegister)`, `BaseAnnotation`.
- New/renamed in `controller`: `DefaultControllerRateLimiter[T comparable]()`, `MustInjectTypeMeta(src, dst)`, `UserFacingError(err, maxLen)`. `EnsureNetworkPolicyForWebhook` now takes `ctx context.Context` first (repo does NOT currently call it — verify with grep; only flag if found).

### 2.2 Multiphase (SSA)
- `multiphase.NewMultiPhaseReconciler(client, name, finalizer, logger, recorder)` — unchanged.
- `multiphase.NewMultiPhaseReconcilerAction(client, conditionName, recorder)` — unchanged.
- `multiphase.NewMultiPhaseStepReconcilerAction[O, S](client, phaseName, conditionName, recorder, fieldManager string)` — **`fieldManager` added, `dryRun` removed**.
- `multiphase.NewMultiPhaseStepReconcilerActionWithDiff[O, S](client, phaseName, conditionName, recorder, fieldManager string)` — diff variant: SSA dry-run classification + `OnDiff` hook.
- `multiphase.NewObjectMultiPhaseStepReconcilerAction[O, src, dst](in MultiPhaseStepReconcilerAction[O, src])` and `...WithDiff` — unchanged generic shape.
- `MultiPhaseStepReconcilerAction` methods now: `Configure(ctx, req, o, logger)`, `Read(ctx, o, data, logger)`, `Diff(ctx, o, read, data, logger)`, `Apply(ctx, o, data, objects, logger)`, `Delete(ctx, o, data, objects, logger)`, `OnSuccess`, `OnError`. **No `ignoreDiff` parameter on `Diff`; no `Create`/`Update` (replaced by `Apply`).**
- `MultiPhaseDiff` adds `GetObjectsToApply()` (union create+update). Keep `AddObjectToCreate/Update/Delete`, `SetObjectsTo*`, `NeedCreate/Update/Delete`, `Diff()`, `IsDiff()`.
- Helpers: `multiphase.ClassifyObjects[T](ctx, c, expected, current, fieldManager string, dryRun bool) (creates, updates, deletes, diffStrs, err)`; `multiphase.PopulateDiff[T](diff, creates, updates, deletes, diffStrs)`; `multiphase.CleanupReadLastAppliedAnnotations[T](ctx, c, read, logger) int` (removes legacy `last-applied-configuration` on managed children).
- SSA requirements: TypeMeta (`apiVersion`/`kind`) must be set on expected objects; deterministic `Name` (no `generateName`); no status fields in expected objects; applies with `client.FieldOwner(fieldManager)` + `client.ForceOwnership`.
- `fieldManager`: use **one stable string for the whole operator**, e.g. `"elasticsearch-operator"`.

### 2.3 Remote (unchanged, 3-way)
- `remote.NewRemoteReconciler(client, name, finalizer, logger, recorder)`, `remote.NewRemoteReconcilerAction(client, recorder)`, `remote.NewRemoteExternalReconciler(handler)` — unchanged.
- Interface adds `GetIgnoresDiff`. Remote `Diff()` still uses `generic-objectmatcher/patch.CalculateOption` + `lastAppliedConfiguration`. Verify exact v3 `RemoteReconcilerAction.Diff` / `RemoteExternalReconciler.Diff` signature at implementation time (§6).

### 2.4 Certificate + workflow (v3.0.6 — redesigned + generic subject/key customization)
The certificate package was redesigned in commit `28f0937e` ("feat: redisign the way fo tls handler", tagged v3.0.4) and further generalized in commit `8f135ed0` ("fix: fix tls saga", tagged **v3.0.6**). Verified against v3.0.6 (one commit ahead of v3.0.4). Surface:

- `certificate.TLSSpec{SecretName, IssuerRef, CommonName, DNSNames, Organization, LeafValidityDays, CAValidityDays, Curve, IPAddresses []string, RenewalDays, GenerateCRL, Subject CertificateSubject, KeyAlgorithm, KeySize, Usages, KeyUsages, CACommonName, CASubject *CertificateSubject}`. (No `SelfSigned`/`CertManager` booleans — backend selection is the operator's construct-time choice.)
- `certificate.CertificateSubject{Organizations, OrganizationalUnits, Countries, Localities, Provinces, StreetAddresses, PostalCodes []string, SerialNumber string}` — the generic RDN block; covers O/OU/Country/Locality/Province (the previously-missing gaps) plus more.
- Key algorithm/size: `KeyAlgorithm` (`KeyAlgorithmECDSA`/`KeyAlgorithmRSA`), `KeySize` (ECDSA 256/384/521; RSA default `DefaultRSAKeySize`=2048, max `MaxRSAKeySize`=8192), ECDSA curve via `Curve` (`CurveP256`/`CurveP384`/`CurveP521`, default P-256), `EffectiveKeyAlgorithm()`/`EffectiveKeySize()`; `Usages`/`KeyUsages` (ExtKeyUsage/KeyUsage vocabularies); `CACommonName`/`CASubject` (CA CN/RDN override). `ValidateContent()`; errors `ErrUnsupportedKeyAlgorithm`/`ErrInvalidKeySize`/`ErrUnknownUsage`/`ErrUnknownCurve`.
- `certificate.TLSBackend[T]` unchanged (`DesiredObjects`, `CertificateSecretName`, `RequiresRotationSaga`). Capability interfaces: `LeafManager[T]`, `NodeSetTLSBackend[T]`, and new `ContentIgnoringBackend` (`IgnoresCertificateContent()` — BYO).
- `certificate.TLSSpecProvider[T]` + `TLSSpecProviderFunc`; new `certificate.CertificateCustomizer[T]` (`CustomizeCertificate(o, base) (TLSSpec, error)`) + `CertificateCustomizerFunc` + `DefaultCertificateCustomizer[T]()` — the generic hook for computing SANs/IPs/CN from cluster state, wired via `rotation.WithCertificateCustomizer`.
- `certificate.LeafChange` + `LeafChangeReason` now includes `LeafSubjectChanged`, `LeafKeyChanged`, `LeafUsagesChanged` (plus Missing/Expiring/CNChanged/OrgChanged/SANsChanged/IPsChanged/NodesChanged/ForceRegen).
- `certificate.RolloutPolicy` (`RolloutOnAdditive` default) + `ShouldRollout` + `RolloutAnnotation` + `LayerSignals` + force annotations — unchanged.
- `selfmanaged`: `BuildCASigner`/`SignLeafSigner`/`ParseCASigner` (generic `crypto.Signer`, RSA+ECDSA), `generateKey`, `CAContentChanged` (CA rotation on subject/key change), `KeyChanged`, `UsagesChanged`; `BuildCA`/`SignLeaf`/`ParseCA` remain as ECDSA wrappers.
- `selfmanaged/pernode.NewPerNodeBackend[T](NodeSpecProvider[T])`, `rotation.NewTLSStep(...)` (now with `WithCertificateCustomizer`), `byo.NewBYOBackend[T]()`, `certmanager.NewCertManagerBackend[T]()`.
- `apis/workflow`/`controller/workflow` — unchanged.
- **Remaining gap (sole item for the dedicated plan `operator-sdk-extra-certificate-customization.md`):** `CARenewalDays` — a CA-specific renewal window. v3.0.6 has `CAValidityDays` (CA validity) and `RenewalDays` (shared CA+leaf renewal); `CANeedsRenewal` still uses the shared `GetValidRenewalDays`. Until it lands, the operator maps `caRenewalDays`→`RenewalDays` (§4.4.4).

---

## 3. Dependency changes (`go.mod`)

Pin **v3.0.7** — already present in the operator's `go.mod` (line 16) and verified good. It carries the cert redesign + generic subject/key customization from v3.0.6 (`CertificateSubject`, `KeyAlgorithm`/`KeySize`/`Curve`, `selfmanaged/pernode`, `rotation.NewTLSStep` + `WithCertificateCustomizer`, `RolloutPolicy.RolloutOnAdditive`).

```diff
-	github.com/disaster37/operator-sdk-extra/v2 v2.0.10
+	github.com/disaster37/operator-sdk-extra/v3 v3.0.7
```

Primary: `go get github.com/disaster37/operator-sdk-extra/v3@v3.0.7`. **No `replace`** — consumed from the proxy.

**Why not `v3.0.3` (still):** `refs/tags/v3.0.3` → `8c9ead49…` = the same commit as `v2.0.10`/`v3.0.1`, module `/v2` / go 1.24 — a mis-tag with no cert work.

**es-handler v8 → v9 + disaster37/elasticsearch client (§4.5.1):**
```diff
-	github.com/disaster37/es-handler/v8 v8.1.5
+	github.com/disaster37/es-handler/v9 v9.0.0
-	github.com/elastic/go-elasticsearch/v8 v8.19.7
-	github.com/olivere/elastic/v7 v7.0.32
+	github.com/disaster37/elasticsearch/v9 v9.0.0   // pulled in (direct or via es-handler)
```

`go mod tidy` drops `github.com/elastic/elastic-transport-go/v8` (transitive via go-elasticsearch) and the olivere transitive deps. **Keep** `github.com/elastic/go-ucfg` (used directly by `pkg/helper/config.go` AND by es-handler v9). **Keep** `kb-handler/v8` + `go-kibana-rest/v8` (Kibana side — unaffected by es-handler v9).

After code changes, run `go mod tidy`. Expected removals if code fully migrated:
- `github.com/disaster37/k8s-objectmatcher v1.8.8` (only if zero remaining imports — see §4.6).

**`goca`:** KEEP only for option (b) interim local backend; DROP once option (a) consumes the library (ECDSA) — see §4.4.4.

Keep (do not remove): `generic-objectmatcher v1.0.2` (remote pattern + es-handler), `go-kibana-rest/v8`, `kb-handler/v8`, `k8sbuilder`, `mergo`, `emperror.dev/errors`, `go-ucfg`.

Version-compatibility note: operator-sdk-extra v3 and es-handler v9 both target `go 1.26` and `controller-runtime v0.19.3`/`k8s.io v0.32.0`; this operator uses `controller-runtime v0.24.1`/`k8s.io v0.36.3` (newer). Go resolves to the operator's versions; SSA client APIs are stable across that range — verify no removed/changed API is hit (R4). The new `disaster37/elasticsearch/v9` client is resty-based and has no controller-runtime/k8s dependency.

---

## 4. Migration design

### 4.1 Mechanical import rename
`/v2/` → `/v3/` in **every** `.go` file under `api/`, `cmd/`, `internal/`, `pkg/` that imports `operator-sdk-extra`. Command (implementation step, do NOT run here):

```
grep -rl 'operator-sdk-extra/v2' --include='*.go' . | xargs sed -i 's#operator-sdk-extra/v2#operator-sdk-extra/v3#g'
```

Then fix compilation incrementally.

### 4.2 Multiphase reconciler constructor changes
For **every** step reconciler (`*_reconciler.go` under the 6 multiphase controller dirs), change:

```go
multiphase.NewMultiPhaseStepReconcilerAction[*X, *Y](
    client, Phase, Condition, recorder,
) // v2 (4 args)
```
to
```go
multiphase.NewMultiPhaseStepReconcilerAction[*X, *Y](
    client, Phase, Condition, recorder,
    "elasticsearch-operator", // fieldManager (single stable name for the whole operator)
)
```

Introduce a shared constant in `internal/controller/common/controller.go`:
```go
const FieldManager = "elasticsearch-operator"
```
and reference `common.FieldManager` from every step reconciler to keep it stable and single-sourced.

For step reconcilers that need the SSA-dry-run "only apply when changed" behavior AND/OR need an `OnDiff` hook (the StatefulSet upgrade-gating reconcilers — see §4.3), use `NewMultiPhaseStepReconcilerActionWithDiff` + `NewObjectMultiPhaseStepReconcilerActionWithDiff` in the controller, and embed `MultiPhaseStepReconcilerActionWithDiff` in the step struct.

### 4.3 StatefulSet reconcilers (custom Diff → SSA + preserved upgrade gating)
Files: `internal/controller/{elasticsearch,logstash,filebeat,metricbeat}/statefulset_reconciler.go`.

Current custom `Diff()` does two things: (a) compute a 3-way patch via `k8s-objectmatcher`, (b) implement "upgrade only one StatefulSet at a time" + TLS-blackout gating using conditions and `data["phase"]`.

Target:
- Change the step struct to embed `multiphase.MultiPhaseStepReconcilerActionWithDiff[...]` and build it with `NewMultiPhaseStepReconcilerActionWithDiff(..., fieldManager)`. Move the "which StatefulSets may be updated now" gating logic into the `OnDiff(ctx, o, data, diff, logger)` hook (available only on the diff variant), which mutates `diff` (e.g., remove objects from `GetObjectsToUpdate()` that must wait) and sets `data["phase"]`.
- Remove `patch.DefaultPatchMaker.Calculate(...)`, `patch.CleanMetadata()`, `patch.IgnoreStatusFields()`, `patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus()`, and the `k8s-objectmatcher/patch` import. Classification is done by the default diff-variant `Diff()` (SSA dry-run). If custom create/update/delete classification is still required, use `multiphase.ClassifyObjects(ctx, client, expected, current, fieldManager, dryRun=true)` + `multiphase.PopulateDiff(...)`.
- Preserve: per-StatefulSet upgrade gating (`localhelper.IsOnStatefulSetUpgradeState`, `*sts.Spec.Replicas==0`, disable/enable routing rebalance via `esHandler`), TLS-blackout branch (delete-all-pods), and `OnSuccess` phase transitions (these are business logic, not diff logic).
- Ensure expected StatefulSets have `TypeMeta` set (builders may already set it; verify — if not, add `APIVersion:"apps/v1", Kind:"StatefulSet"`).

### 4.4 Certificate lifecycle (full v3 adoption)

Replace the custom **condition-based FSM** with the v3 `workflow` saga; keep the CRD spec unchanged (one additive field); keep cert generation **local** (behind `certificate.TLSBackend`) so O/OU/SAN/CN/IP are computed exactly as today. The stock `selfmanaged` backend is NOT the generator (see §2.4).

**4.4.1 CRD API — unchanged except TWO additive optional fields (`caRenewalDays`, `KeyComplexity`).**
- Do NOT remove/rename/retype any existing spec field (`TlsSpec.Enabled/SelfSignedCertificate/CertificateSecretRef/ValidityDays/RenewalDays/KeySize`, logstash `Pki.*`, `FilebeatPkiSpec.*`). `KeySize` is **retained but deprecated**: it is only honored when `KeyComplexity` is empty (backward-compatible default).
- Add TWO additive optional fields:
  ```go
  // CaRenewalDays is the number of days before the CA certificate expires
  // at which the operator starts the CA rotation saga. Default 30.
  // +operator-sdk:csv:customresourcedefinitions:type=spec
  // +optional
  // +kubebuilder:default=30
  CaRenewalDays *int `json:"caRenewalDays,omitempty"`

  // KeyComplexity selects the private-key algorithm and strength:
  // rsa-2048 (default), rsa-4096, ecdsa-p256, ecdsa-p384, ecdsa-p521.
  // Empty => legacy KeySize applies (RSA, default 2048). Changing it triggers
  // a full CA rotation + staged cluster-wide rolling restart (no mixed
  // RSA/ECDSA cluster). See §4.4.4/§4.4.6.
  // +operator-sdk:csv:customresourcedefinitions:type=spec
  // +kubebuilder:validation:Enum=rsa-2048;rsa-4096;ecdsa-p256;ecdsa-p384;ecdsa-p521
  // +optional
  KeyComplexity string `json:"keyComplexity,omitempty"`
  ```
  Placement: `api/shared/tls.go` `TlsSpec` (covers elasticsearch + kibana, both `Spec.Tls`). Parallel additive fields in `api/logstash/v1/logstash_types.go` Pki spec and `api/beat/v1/filebeat_types.go` `FilebeatPkiSpec` **only if** those single-cert apps should support CA renewal + complexity selection (default: out-of-scope; keep their existing `KeySize`/`RenewalDays` behavior).
- Deepcopy/manifests impact: `make generate manifests` regenerates `zz_generated.deepcopy.go` and adds the fields to `config/crd/bases/*.yaml` (incl. the `Enum` validation on `keyComplexity`). Additive + optional → no breaking change to existing CRs.

**4.4.2 Status-only workflow cursor (decision: keep it in status).**
The saga needs a durable cursor. Keep `tlsWorkflowStatus` as a **status-subresource** addition — status is operator-owned, not user-authored, so this does NOT alter the user-facing spec API:
- Add to `ElasticsearchStatus`/`LogstashStatus`/`KibanaStatus`/`FilebeatStatus`:
  ```go
  // TlsWorkflowStatus tracks the CA rotation saga phase.
  // +optional
  TlsWorkflowStatus workflow.WorkflowStatus `json:"tlsWorkflowStatus,omitempty"`
  ```
  (import `github.com/disaster37/operator-sdk-extra/v3/pkg/apis/workflow`).
- Add `func (s *XStatus) GetWorkflowStatus() *workflow.WorkflowStatus { return &s.TlsWorkflowStatus }` in the corresponding `*_func.go` to satisfy `controller/workflow.WorkflowStatusGetter`.
- Rationale vs alternatives: status is durable in etcd (survives restart), is the idiomatic v3 mechanism (`WorkflowStatus` embed), and is additive; an annotation-on-CR or ConfigMap cursor would be non-idiomatic and need extra ownership handling. (Verify where v3 asserts `GetWorkflowStatus()` — object vs status — in `workflow_step_action.go`; see R7.)

**4.4.3 O/SAN/CN auto-computation — preserve exactly (mapping table).**
The controller keeps computing every cert attribute from topology (no user input). Today's `goca.Identity` fields map onto the local backend's inputs:

| Cert | CN | O | OU | DNSNames (auto) | IPs | Validity | Key |
|---|---|---|---|---|---|---|---|
| ES transport CA | `<name>-transport` | `<name>` | `transport` | — | — | `ValidityDays||397` | `KeySize||2048` |
| ES transport node `<node>` | `<nodeName>` | `<name>` | `<nodeGroupName>` | node headless/global svc + `nodeName` + `.ns`/`.ns.svc` | `127.0.0.1` | `ValidityDays||397` | `KeySize||2048` |
| ES API CA | `<name>-api` | `<name>` | `api` | — | — | `ValidityDays||397` | `KeySize||2048` |
| ES API cert | `<name>` | `<name>` | `api` | global svc + node-group svc + `AltNames` | `AltIps` | `ValidityDays||397` | `KeySize||2048` |
| Logstash CA | `<name>-logstash` | `<name>` | `logstash` | — | — | `ValidityDays||365` | `KeySize||2048` |
| Logstash cert `<key>` | `<key>` | `<key>` | `logstash` | service/ingress names + `AltNames` | `AltIps` | `ValidityDays||365` | `KeySize||2048` |

- `Country/Locality/Province = internal` on all certs — preserve.
- These map onto `certificate.TLSSpec` / `certificate.CertificateSubject` / `KeyAlgorithm`/`KeySize`/`Curve` in v3.0.7 (O→`Subject.Organizations`, OU→`Subject.OrganizationalUnits`, Country/Locality/Province→`Subject.Countries/Localities/Provinces`, SANs→`DNSNames`/`IPAddresses`, CN→`CommonName`/`CACommonName`, key→`KeyAlgorithm`+`Curve`/`KeySize`). All now representable in the library — no local backend needed (see §4.4.4 option a).
- **ECDSA/complexity:** `KeyComplexity` maps to `TLSSpec.KeyAlgorithm` + (`TLSSpec.Curve` for ECDSA | `TLSSpec.KeySize` for RSA): `ecdsa-p256/p384/p521` → `KeyAlgorithm=ECDSA` + `Curve=P-256/P-384/P-521`; `rsa-2048/4096` → `KeyAlgorithm=RSA` + `KeySize=2048/4096`; empty → legacy `KeySize` (RSA 2048). ECDSA is supported by ES/Logstash/Kibana (see §4.4.4.1).

**4.4.4 Certificate generation: consume the library (option a) vs interim local backend (option b).**

The library (v3.0.7) now generates IP SANs, ECDSA/RSA keys (`KeyAlgorithm`/`KeySize`), full subject RDN (`CertificateSubject`: O/OU/Country/Locality/Province), CA/leaf validity + renewal, per-node certs, and the rotation saga — so the ES operator should NOT build a from-scratch backend. Two ordered options:

- **Option (a) — RECOMMENDED: consume the library now.** Implement only the ES-specific `certificate.TLSSpecProvider[*X]` + `selfmanaged/pernode.NodeSpecProvider[*X]` + `rotation.WithCertificateCustomizer` (the O/OU/SAN/CN/IP auto-computation from §4.4.3, currently in `secret_tls_builder.go`), and drive everything with `rotation.NewTLSStep` + `selfmanaged/pernode.NewPerNodeBackend` (transport) + `selfmanaged.NewSelfManagedBackend` (API) + `byo.NewBYOBackend`. `goca`/`pkg/pki`/`secret_tls_builder.go` are then DELETED. v3.0.7 covers OU/Country/Locality/Province (`CertificateSubject`), RSA (`KeyAlgorithm=RSA` + `KeySize`), and ECDSA curve selection (`Curve`), so `KeyComplexity` maps 1:1 (§4.4.3). The only item still pending is `CARenewalDays` (below).
- **Option (b) — interim local backend (use only if option (a) is deferred).** Refactor `secret_tls_builder.go`'s builders into a local type implementing `certificate.TLSBackend[*X]` + `NodeSetTLSBackend[*X]` + `LeafManager[*X]` (goca retained), then drive it with `rotation.NewTLSStep`. Now nearly redundant — option (a) covers everything except `CARenewalDays` — so this is only a fallback if the operator must retain a CA-specific renewal window before the library ships it.

**Reconciliation of `caRenewalDays`:** the ES CRD keeps its user-facing `caRenewalDays` field (§4.4.1). v3.0.7 has no CA-specific renewal window (`CAValidityDays` is CA *validity*; `RenewalDays` is the *shared* CA+leaf renewal window used by `CANeedsRenewal`), so option (a) maps `caRenewalDays`→`TLSSpec.RenewalDays` (shared window) for now; option (b) honors it exactly in the local backend. Once the library adds `CARenewalDays` (dedicated plan), map `caRenewalDays`→`TLSSpec.CARenewalDays`. No double-adding: `CARenewalDays` is added to the **library**, `caRenewalDays` stays in the **operator CRD**.

**RSA→ECDSA migration (complexity change = CA rotation).** A change to `KeyComplexity` changes `TLSSpec.KeyAlgorithm`/`Curve`/`KeySize`; v3.0.7 detects this via `CAContentChanged` (CA) and `LeafNeedsChange`→`LeafKeyChanged` (leaf), so the saga enters `""→Rotate` **even if the CA is not near expiry** — exactly like a CA renew: emit a new CA + all leaves under the new algorithm, bundle the old CA (`ca.crt = newCA||oldCA`), gate on `WaitForOwnedObjects` (full staged rolling restart, one StatefulSet at a time), then `Converge` strips the old CA. **No mixed RSA/ECDSA cluster is ever allowed** — the whole cluster rolls together, matching the "rolling restart whole cluster like a CA renew" expectation.

**4.4.4.1 ECDSA compatibility (per-product verdict, current 8.x).** ECDSA certificates are supported for TLS by all three products; the operator targets ES 8.x (`docker.elastic.co/elasticsearch/elasticsearch:<version>`, `es-handler/v9` + `disaster37/elasticsearch/v9`), Logstash 8.x, Kibana 8.x.
- **Elasticsearch** — `xpack.security.transport.ssl`/`http.ssl` accept PEM keys with no algorithm restriction; the default `ssl.cipher_suites` includes `TLS_ECDHE_ECDSA_*` (ECDSA certs supported at runtime). The bundled JDK supports P-256/P-384/P-521. Caveat: `elasticsearch-certutil` in 8.x only exposes RSA `--keysize` (no `--curve`), but that helper-tool limitation does not affect runtime cert acceptance (the operator generates certs itself). Docs: [certutil](https://www.elastic.co/docs/reference/elasticsearch/command-line-tools/certutil), [security settings](https://www.elastic.co/docs/reference/elasticsearch/configuration-reference/security-settings) (`ssl.cipher_suites`, `ssl.key`, `ssl.supported_protocols`).
- **Kibana** — `server.ssl` uses Node.js/OpenSSL → full ECDSA P-256/P-384/P-521 support (no restriction). Docs: [Kibana security settings](https://www.elastic.co/docs/reference/kibana/configuration-reference/security-settings) / `server.ssl.*`.
- **Logstash** — JRuby + `elasticsearch` output (Java via Manticore/JDK) → ECDSA supported. Docs: [Logstash secure settings](https://www.elastic.co/docs/reference/logstash/configuration-reference/secure-settings).
- **Go clients** (es-handler, kb-handler, filebeat→ES output): Go `crypto/tls` always supports ECDSA; the operator's ES HTTP client uses `InsecureSkipVerify: true` and beats trust via `certificate_authorities` (CA) — both algorithm-agnostic; no RSA pinning in code.
- **Recommended default curve: P-256** (`ecdsa-p256`). P-384/P-521 are supported by the bundled JDK/OpenSSL but P-521 has historically been the least-tested on some JVM/FIPS provider combinations — keep it available but not default.

**4.4.5 ES transport TLS — shared CA + per-node leaf; NO cluster-wide reconcile on membership change.**
Decision on "one Secret per node": evaluated and **not adopted as the mount mechanism**, because a StatefulSet pod-template volume requires a static `secretName` — it cannot vary per pod (N static volumes would churn the template on every scale change and force a full rollout, defeating the goal). The current shared-Secret + `${POD_NAME}.crt` selection is the correct mount. The per-node requirement is satisfied at the **data/logic** level:

- **Shared transport CA Secret** `<name>-pki-transport-es` (keep existing name): `ca.crt`/`ca.key`/`ca.pub`/`ca.crl`.
- **Shared transport leaf Secret** `<name>-tls-transport-es` (keep): `ca.crt` + per-node `<node>.crt`/`<node>.key`. Mounted as today (`node-tls` volume; init-filesystem copies `${POD_NAME}`'s files).
- **Node ADD (scale-up):** the tls step emits the new node's `<node>.crt`/`.key` into the shared Secret via SSA **without changing the rollout marker** → existing pods do not restart; the new pod reads its cert at init (today's behavior preserved).
- **Node REMOVE (scale-down):** the tls step removes that node's keys via SSA **without changing the rollout marker** → no cluster-wide reconcile (today's behavior preserved).
- **Leaf renew / CA rotate:** bump the rollout marker so the StatefulSet rolls, staged by §4.3's upgrade-gating (one StatefulSet at a time). Where a single-node restart is desired, delete that pod directly (StatefulSet recreates it and re-runs init to copy the updated cert) — see R1 trade-off.
- **Rollout marker:** keep the `.../sequence`-style marker (rename semantics to "rotation marker", bumped ONLY on intentional CA/leaf rotation) rather than `certificate.SecretHash` over the whole Secret — hashing the whole Secret would change on node add/remove and re-introduce full rollouts. Optionally expose the marker via `certificate.SecretHashAnnotation` applied to the **rotation-triggering** subset (CA cert + counter). Membership changes MUST NOT feed the template hash.

**4.4.5.1 Trust-model verification (node-add-without-rollout guarantee).**
Verified `xpack.security.*` settings in `internal/controller/elasticsearch/configmap_builder.go` and `statefulset_builder.go`:
- Transport: `xpack.security.transport.ssl.enabled=true`, `verification_mode=full`, `certificate=.../transport-cert/${POD_NAME}.crt`, `key=.../${POD_NAME}.key`, `certificate_authorities=.../transport-cert/ca.crt`.
- HTTP (when TLS enabled): `xpack.security.http.ssl.certificate=.../api-cert/tls.crt`, `key=tls.key`, `certificate_authorities=ca.crt`.
- **No `truststore` config and no static peer-certificate list anywhere** (grep for `truststore` / peer cert lists across the repo, `pkg/pki`, and the es-handler client returns nothing). `ca.crt` reaches every node via the shared `node-tls` Secret volume (init-filesystem copies it into `transport-cert/`).
- `network.publish_host` = pod name (`metadata.name`) in `statefulset_builder.go`, and each node cert's SANs include that pod name + headless/global service DNS names (auto-generated, §4.4.3).

**Verdict: adding a node without rolling existing nodes CANNOT break trust**, because (1) trust is anchored on the shared CA (`ca.crt`), identical for all nodes and unchanged by node add (only new `<node>.crt`/`.key` keys are added); (2) `verification_mode=full` only requires the peer cert to be CA-signed AND its SAN to match the address the peer is dialed by (the pod-name publish host, already in the SAN list); (3) there is no per-node cert list to refresh. The invariant to preserve: per-node SAN generation MUST keep covering each new node's pod name/DNS (already guaranteed by §4.4.3) — never switch trust to a static peer list, and never let node add/remove feed the rollout marker (§4.4.5).

**4.4.6 Workflow saga (ES, replaces the condition FSM) + blackout + bootstrap.**
Drive `secret_tls_reconciler.go` (elasticsearch) with the library's `rotation.NewTLSStep(...)` (embeds `workflow.DefaultWorkflowStepReconcilerActionWithDiff`, phases `""→Rotate→Converge→""`), passing the per-node backend + provider, `WithConvergenceCheck` (the `WaitForOwnedObjects` gate from §4.3), and `WithLabelsDecorator`/`WithAnnotationsDecorator` (`getLabels`/`getAnnotations`). The step publishes `certificate.LayerSignals` under `data["tls.<phase>"]`; the statefulset step consumes it with `certificate.ShouldRollout(RolloutOnAdditive, sig)` to decide the rollout marker (additive-only: node add/remove and SAN-removal do NOT roll).

- **Bootstrap (initial cluster):** phase empty + `!o.IsBoostrapping()` + missing/expired CA → the `""` phase emits CA + all node leaves + API cert (the leaf-only path is skipped when the CA is missing); keep `o.Status.IsBootstrapping` semantics unchanged (`configmap_builder.go` already gates `cluster.initial_master_nodes` on it).
- **CA rotation (expiry OR content change):** `selfmanaged.CANeedsRenewal` (expiry) **or `selfmanaged.CAContentChanged` (subject/key-algorithm/size change — e.g. `KeyComplexity` RSA→ECDSA)** drives `""→Rotate`; `Rotate` emits new CA + leaf with `ca.crt = newCA||oldCA` bundle; the `ConvergenceCheck` gates until all StatefulSets have rolled (`WaitForOwnedObjects`); `Converge` strips the old CA.
- **Leaf renewal:** `LeafNeedsChange` drives leaf-only regen via `LeafManager.DesiredLeafWithCA` (no new CA, no phase write) — maps to today's `renew-certificates`/`RenewalDays` behavior.
- **Blackout (ES-specific, kept):** when certs are expired / the saga stalled (`transportRootCA.NotAfter < now` or any node leaf expired) while `o.IsBoostrapping()` — set a user-facing `TlsBlackout` condition and force full regeneration + delete-all-pods recovery (the existing `TlsConditionBlackout` branches in `statefulset_reconciler.go`/`secret_tls_reconciler.go`, layered on top of the saga).

**Logstash / kibana / filebeat (single-cert apps):** keep their simpler regenerate-on-expiry model; optionally wrap in a minimal single Generate/Propagate cycle if CA rotation needs staging — by default keep existing behavior, swap the FSM for the workflow step action only. Preserve `Spec.Pki`/`Spec.Tls`/`FilebeatPkiSpec` and per-name cert generation.

**4.4.7 BYO (user-provided secret).**
When `TlsSpec.CertificateSecretRef` is set (or `Pki` equivalent), use `byo` semantics: emit no child objects; validate the referenced Secret exists and wire it as today (`GetSecretNameForTlsApi` returns the user secret name). No saga (`RequiresRotationSaga()==false`).

**Remove (revised):**
- The condition-FSM machinery: `TlsGeneratePki/TlsPropagatePki/TlsGenerateCertificate/TlsPropagateCertificate` conditions and `TlsPhaseUpdatePki/PropagatePki/UpdateCertificates/PropagateCertificates/CleanTransportCA/Reconcile` internal phases (replaced by `WorkflowStatus.CurrentPhase`). Keep `TlsReady` + `TlsBlackout` as user-facing conditions.
- `k8s-objectmatcher` usage in `secret_tls_reconciler.go` ×4 (SSA replaces `patch.DefaultAnnotator.SetLastAppliedAnnotation`).
- `pkg/helper/diff.go` (label/annotation/ownerRef diff) — SSA + the saga make it dead (§4.6).
- **Do NOT remove** `secret_tls_builder.go` / `pkg/pki` / `goca` under option (b) interim backend; under option (a) **DELETE** them (generation replaced by `selfmanaged`/`selfmanaged/pernode` + `TLSSpecProvider`/`NodeSpecProvider`).

### 4.5 Remote controllers
- **es-handler v8 → v9**: the Elasticsearch-side remote controllers (`internal/controller/elasticsearchapi`) need a full client/type migration — see §4.5.1. The Kibana-side controllers (`internal/controller/kibanaapi`) are unaffected (kb-handler/go-kibana-rest unchanged).
- Confirm the v3 `remote.RemoteReconcilerAction`/`RemoteExternalReconciler` `Diff()`/`GetIgnoresDiff` signatures still use `generic-objectmatcher/patch.CalculateOption` (they should; remote is unchanged). If the signature changed, update the `Diff(... ignoresDiff ...patch.CalculateOption)` methods in `*_reconciler.go` and `*_api_client.go` accordingly.
- `api/*/v1/*_types.go` importing `apis`/`apis/remote` and `*_func.go` importing `object` — mechanical rename only.

### 4.5.1 es-handler v8 → v9 + disaster37/elasticsearch client migration

The maintainer released `es-handler v9.0.0`, which replaces BOTH Elasticsearch clients (official `go-elasticsearch/v8` transport + `olivere/elastic/v7` types) with a single new client `github.com/disaster37/elasticsearch/v9` (resty-based). **`ElasticsearchHandler` method names/semantics are unchanged; only the client config type and the API-object types change.**

**New library facts (verified @ v9.0.0):**
- `es-handler/v9` (`module github.com/disaster37/es-handler/v9`, go 1.26): `ElasticsearchHandler` interface unchanged in method set; `Client()` now returns `elasticsearch.Client` (interface); `NewElasticsearchHandler(cfg *elasticsearch.Config, log *logrus.Entry)`.
- `elasticsearch/v9` root (`module github.com/disaster37/elasticsearch/v9`, go 1.26): `elasticsearch.New(cfg *Config, log) (Client, error)`; `Config{URL string, Username, Password, APIKey, BearerToken string, TLSSkipVerify, AllowInsecureHTTP bool, CACert []byte, Timeout, IdleConnTimeout time.Duration, DisableHTTP2 bool, RetryCount int, ...}` — **single `URL`, no `Addresses`, no `Transport`**. Error helpers `IsNotFound/IsConflict/IsUnauthorized`.
- `api` package: services (`Cluster`, `Indices`, `Ingest`, `Snapshot`, `SLM`, `ILM`, `Transform`, `Security`, `Watcher`, `License`, `Info`) — every method takes `ctx` first; non-2xx → `*types.ElasticsearchError`.

**Type mapping (operator-relevant):**

| Resource | v8 type | v9 type |
|---|---|---|
| License Get/Diff | `olivere.XPackInfoLicense` | `esapi.LicenseInfo` |
| ILM | `olivere.XPackIlmGetLifecycleResponse` | `esapi.IlmPolicy` |
| SnapshotRepository | `olivere.SnapshotRepositoryMetaData` | `esapi.SnapshotRepository` (Settings `map[string]string`) |
| RoleMapping | `olivere.XPackSecurityRoleMapping` | `esapi.SecurityRoleMapping` |
| User create/update/diff | `olivere.XPackSecurityPutUserRequest` | `eshandler.SecurityPutUserRequest` (new local) |
| User get | `olivere.XPackSecurityUser` | `esapi.SecurityUser` (`FullName`, was `Fullname`) |
| ComponentTemplate | `olivere.IndicesGetComponentTemplate` | `eshandler/patch.ComponentTemplate` (new) |
| IndexTemplate | `olivere.IndicesGetIndexTemplate` | `eshandler/patch.IndexTemplate` (new) |
| Watch | `olivere.XPackWatch` | `eshandler.XPackWatch` (= `map[string]any`) |
| IngestPipeline | `olivere.IngestGetPipeline` | `esapi.IngestPipeline` |
| ClusterHealth | `olivere.ClusterHealthResponse` | `esapi.ClusterHealthResponse` |
| Role | KEEP `eshandler.XPackSecurityRole` | KEEP |
| SLM | KEEP `eshandler.SnapshotLifecyclePolicySpec` | KEEP |
| Transform | KEEP `eshandler.Transform` | KEEP |

**Per-file changes:**

- `internal/controller/elasticsearchapi/helper.go` — the biggest change: replace `elastic.Config{Transport, Addresses, Username, Password, Logger}` + `elastictransport.JSONLogger` with `elasticsearch.Config{URL, Username, Password, TLSSkipVerify}`. `hosts []string` (multiple) collapses to a single `URL`: managed → the single service URL; external → `ExternalElasticsearchRef.Addresses[0]` (log a warning if >1). Map `selfSignedCertificate` → `TLSSkipVerify: true`. Drop `http.Transport`/`net.Dialer`/`elastictransport`.
- `internal/controller/elasticsearch/elasticsearch_controller.go` — `getElasticsearchHandler` builds `elastic.Config{Addresses, Username, Password, Transport}` → `elasticsearch.Config{URL, Username, Password, TLSSkipVerify: true}` (it already sets `InsecureSkipVerify`). `eshandler` import `/v8` → `/v9`.
- `internal/controller/elasticsearch/statefulset_reconciler.go` — `eshandler` import `/v8` → `/v9` only (uses `ElasticsearchHandler` interface + `EnableRoutingRebalance`/`DisableRoutingRebalance`, unchanged).
- `internal/controller/elasticsearchapi/*_api_client.go` (10 files) — change `Build`/`Get`/`Diff` olivere types → v9 types per the table; update `eshandler` import; `Diff` still delegates to `h.Client().*Diff(...)`.
- `internal/controller/elasticsearchapi/*_controller.go` (10 files) — update generic type params `remote.RemoteReconciler[*X, *olivere.T, eshandler.ElasticsearchHandler]` → `*v9type`.
- `internal/controller/elasticsearchapi/*_reconciler.go` (10 files) — update `remote.RemoteRead[*olivere.T]` / `RemoteDiff[*olivere.T]` → v9 types; the `ignoreDiff ...patch.CalculateOption` stays (generic-objectmatcher).
- `api/elasticsearchapi/v1/{componenttemplate,indextemplate,indexlifecyclepolicy}_webhook.go` — replace olivere JSON-unmarshal validation targets (`olivere.IndicesGetComponentTemplate`/`IndicesGetIndexTemplate`/`XPackIlmGetLifecycleResponse`) with `eshandler/patch.ComponentTemplate`/`patch.IndexTemplate`/`esapi.IlmPolicy` (or a plain `map[string]any` sanity check).
- `internal/controller/elasticsearchapi/*_controller_test.go` + `suite_test.go` — `es-handler/v8/mocks` → `es-handler/v9/mocks`; olivere test fixtures → v9 types.

**Edge cases / error handling:**
- 404 semantics unchanged at the es-handler layer (Get→`(nil,nil)`, Delete→idempotent); the operator's api_clients already wrap ES 404 as "resource not found" through the remote reconciler.
- Single-URL constraint: external ES with multiple `Addresses` loses multi-host failover (new client takes one URL). Decide: use `Addresses[0]` (documented limitation) or reject >1.
- Debug logging: v8 used `elastictransport.JSONLogger`; v9 logs via the injected `*logrus.Entry` — drop the JSONLogger block.
- Auth: v8 used `Username`/`Password`; v9 also supports `APIKey`/`BearerToken` (not needed now).
- Go 1.26 toolchain: es-handler v9 + elasticsearch/v9 require `go 1.26` (operator already 1.26.0). OK.
- The operator does **not** call `ClusterGetSettings`/`ClusterPutSettings`/`GetVersion` (verify with grep) — those v8 methods are absent from v9's interface; confirm no usage before compiling.

**Test strategy:** envtest suites in `internal/controller/elasticsearchapi` use `es-handler/v8/mocks` (`MockElasticsearchHandler`) — switch to `es-handler/v9/mocks`; update fixture types. Add a smoke assertion that `GetElasticsearchHandler` builds a handler from a managed ES CR and from an external secret (single-URL path).

### 4.6 diffIgnore / diff-function decision (remove + document)
**Remove** (SSA makes client-side K8s diff obsolete):
- `pkg/helper/diff.go` + `pkg/helper/diff_test.go` (`DiffLabels`, `DiffAnnotations`, `DiffOwnerReferences`).
- All `k8s-objectmatcher/patch` imports and calls in multiphase reconcilers and their `*_test.go` (replace test assertions that used `patch.DefaultPatchMaker.Calculate` with SSA-based assertions or builder-value assertions).
- `github.com/disaster37/k8s-objectmatcher` from `go.mod` once unused.

**Keep** (different mechanism, not SSA — document why):
- `generic-objectmatcher/patch` for remote (ES/Kibana API) 3-way merge in `*_api_client.go` and `third_party/es-handler/*`. Rationale: the remote pattern reconciles **external API objects**, not K8s objects; SSA field ownership does not apply, and v3's remote pattern still uses `lastAppliedConfiguration` + 3-way merge.

**Documentation file to create** (implementation step): `documentations/ssa-and-diff-removal.md` with outline:
1. What changed: v2 client-side 3-way strategic-merge (`k8s-objectmatcher`) → v3 Server-Side Apply with `fieldManager` + `ForceOwnership`.
2. Why `DiffPathOption`/`DiffFunc`/`IgnoreUnset`/`ignoreDiff ...patch.CalculateOption` and `patch.DefaultAnnotator.SetLastAppliedAnnotation` were removed: SSA keeps per-field ownership in `metadata.managedFields`; fields the operator does not set are simply not applied, so "ignore field" options are unnecessary; `last-applied-configuration` annotation is no longer used by multiphase/sentinel patterns.
3. Fields the operator must not own: omit them from expected objects; do not set them; SSA leaves other managers' fields intact. Residual conflicts: `Apply(ForceOwnership)` resolves; `Conflict` errors are surfaced and retried by controller-runtime backoff.
4. Why `generic-objectmatcher` (remote pattern) is retained: external API 3-way merge, not SSA; list the files that still use it.
5. Upgrade path: existing objects previously managed via update/merge + `last-applied-configuration` are re-owned on first SSA apply (ForceOwnership); `multiphase.CleanupReadLastAppliedAnnotations` removes the legacy annotation from managed children.

---

## 5. Rollout order (ordered steps with definition-of-done + verification)

Run `go mod tidy` early and re-run after each step. `go build ./...`, `go vet ./...`, `go fmt ./...` must pass at the end of every step.

1. **Dependency bump.** `go get github.com/disaster37/operator-sdk-extra/v3@v3.0.7` + `go get github.com/disaster37/es-handler/v9@v9.0.0` (+ `github.com/disaster37/elasticsearch/v9@v9.0.0`); drop `es-handler/v8`, `go-elasticsearch/v8`, `olivere/elastic/v7`. *DoD:* `go.mod`/`go.sum` updated; `go list -m all` shows v3.0.7 + es-handler v9.0.0; build broken (expected) — module graph only.

2. **es-handler v9 client/type migration (§4.5.1).** Update `internal/controller/elasticsearchapi/helper.go`, `internal/controller/elasticsearch/elasticsearch_controller.go`, `elasticsearch/statefulset_reconciler.go`, all `elasticsearchapi/*_api_client.go`/`*_controller.go`/`*_reconciler.go`, the 3 `api/elasticsearchapi/v1/*_webhook.go`, and the `*_test.go`/`suite_test.go` mocks. *DoD:* no remaining `es-handler/v8`, `go-elasticsearch/v8`, `olivere/elastic`, `elastic-transport-go` imports; `go mod tidy` drops them.

3. **Mechanical import rename** `/v2/` → `/v3/` across the repo (operator-sdk-extra; grep+sed). *DoD:* no remaining `/v2/` imports; `go build ./...` reports **only** API-signature errors (constructor arg counts, missing `Apply`, `Diff` signatures), no unknown-package errors.

4. **Multiphase constructor + `fieldManager`.** Add `common.FieldManager` const; thread `fieldManager` into every `NewMultiPhaseStepReconcilerAction`. For statefulset reconcilers, switch to `...WithDiff` variant + `OnDiff`. *DoD:* `go build ./...` clean for multiphase packages (excluding cert TLS packages still in progress).

5. **SSA migration of non-TLS step reconcilers.** Remove `k8s-objectmatcher` from `statefulset_reconciler.go` ×4 and `pdb_reconciler.go`; ensure builders set `TypeMeta` on expected objects; update `*_controller_test.go`/`*_builder_test.go` that referenced `patch.*`. *DoD:* `go build ./...` + `go vet ./...` clean; `go test ./internal/controller/{elasticsearch,logstash,filebeat,metricbeat,cerebro}/...` (with envtest) green.

6. **Certificate lifecycle (full adoption).** Add `caRenewalDays` + `KeyComplexity` (+ optional `TlsWorkflowStatus` status) to the CRDs; implement the ES-specific `TLSSpecProvider`/`NodeSpecProvider` and drive the 4 `secret_tls_reconciler.go` via the library's `rotation.NewTLSStep` + `selfmanaged/pernode`/`selfmanaged`/`byo` (option a) — or, if deferred, the interim local goca backend (option b). Replace the condition FSM with the saga (ES: Rotate→Converge + bootstrap + blackout, incl. `CAContentChanged` for RSA→ECDSA); adopt `ShouldRollout(RolloutOnAdditive)` so node add/remove does NOT roll. *DoD:* `go build ./...`/`go vet ./...` clean; `go mod tidy` drops `k8s-objectmatcher` (and `goca` under option a); TLS/cert envtest suites green (`TEST=true`); lifecycle + trust-model + RSA→ECDSA cases (§7) pass.

7. **Remove dead diff code + docs.** Delete `pkg/helper/diff.go` + `diff_test.go`; confirm zero `k8s-objectmatcher` imports; write `documentations/ssa-and-diff-removal.md`. *DoD:* `go mod tidy` drops `k8s-objectmatcher`; doc exists with §4.6 outline.

8. **Remote controllers + API types.** Mechanical `/v2/`→`/v3/` + es-handler v9 already done in steps 2–3; now resolve any `remote` package signature drift (verify `Diff`/`GetIgnoresDiff`). *DoD:* `go build ./...` clean for `internal/controller/elasticsearchapi`, `internal/controller/kibanaapi`, `api/**`.

9. **Regenerate + full test.** Run `make generate manifests` (controller-gen v0.21.0; bump if v3 forces newer controller-tools), `make fmt`, `make vet`, and `make test` (envtest, `ENVTEST_K8S_VERSION=1.25.x`, `TEST=true`). *DoD:* deepcopy/manifests regenerated with no unexpected diffs (review `git diff` on `zz_generated.deepcopy.go` and `config/crd/bases`); full `make test` green; `make build` green.

Verification commands per step: `go build ./...`, `go vet ./...`, `go fmt ./...`, `go test -p 1 -timeout 1200s -count 1 ./apis/... ./pkg/... ./controllers/...` (or `make test` for envtest), `go mod tidy`, `go list -m all | grep operator-sdk-extra`.

---

## 6. Risks and unknowns (with fallback instructions)

- **R1 — Per-node rollout granularity (highest risk, but trust-model verified safe).** Per-node Secrets cannot be mounted by a StatefulSet pod template (static `secretName`), so leaf certs stay in the shared per-nodegroup Secret selected by `${POD_NAME}.crt`. Trust is CA-anchored (§4.4.5.1), so node add/remove without rollout is SAFE; the library's `RolloutOnAdditive` policy encodes exactly this (node add/remove → no rollout). Remaining limitation: a leaf/CA rotation that changes the pod template rolls ALL pods of the nodegroup; targeted single-node restart requires deleting the specific pod. *Trade-off to validate in envtest/e2e:* pod deletion vs staged full rolling restart (§4.3 gating). **Gate:** confirm scale-up/scale-down does NOT roll unchanged pods and that new-node certs are trusted before merging.
- **R2 — `remote` package signature drift.** Migration guide implies remote is unchanged, but `GetIgnoresDiff` is new. *Fallback:* `go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote.RemoteReconcilerAction` and `...RemoteExternalReconciler`; compare against current overrides; adjust signatures.
- **R3 — Remaining library gap (`CARenewalDays`, a CA-specific renewal window).** v3.0.7 now covers IP SANs, ECDSA/RSA keys (`KeyAlgorithm`/`KeySize`), full `CertificateSubject` (O/OU/Country/Locality/Province), CA/leaf validity + renewal, per-node backend, and the rotation saga — but has no CA-specific renewal window (`CAValidityDays` is CA *validity*; `RenewalDays` is the *shared* CA+leaf window used by `CANeedsRenewal`). *Resolution:* option (a) maps `caRenewalDays`→`RenewalDays` (shared window) until the dedicated library plan (`operator-sdk-extra-certificate-customization.md`) adds `CARenewalDays`; option (b) honors it exactly. Either way the dedicated plan is the eventual fix.
- **R4 — controller-runtime/k8s.io version skew.** operator-sdk-extra v3 (v3.0.7) and es-handler v9 both target controller-runtime v0.19.3 / k8s.io v0.32.0; this operator uses v0.24.1 / v0.36.3 (newer). Go resolves to the operator's versions; SSA client APIs (`client.Apply`/`FieldOwner`/`ForceOwnership`) are stable across that range but verify no removed/changed API is hit when building against the pinned v3.0.7. *Fallback:* `go build ./...` + `go vet ./...` immediately after the pin; if an API mismatch surfaces, patch the local library shim or pin a controller-runtime version compatible with both. Re-run `make generate manifests` and bump `CONTROLLER_TOOLS_VERSION` if controller-gen incompatibility arises.
- **R5 — helper package surface.** Functions like `GetZapLogLevelFromEnv`, `ToSlicePtr`, `ToSliceOfObject`, `DiffMapString`, `RandomString`, `HasCRD`, `Get`, `PrintVersion`, `GetKubeClientTimeoutFromEnv` are assumed unchanged. *Fallback:* `go doc github.com/disaster37/operator-sdk-extra/v3/pkg/helper` and grep for renamed/removed helpers; use `helper/ssa` subpackage if it provides SSA helpers.
- **R6 — `object`/`apis` surface.** `object.MultiPhaseObjectStatus`/`RemoteObjectStatus`, `apis.MapAny`, `apis/shared` names assumed unchanged. *Fallback:* `go doc` each; only import-path renames expected.
- **R7 — `workflow` status-getter wiring.** v3 reads the phase cursor via `WorkflowStatusGetter` on the object/status; exact type-assertion target is uncertain. *Fallback:* read `pkg/controller/workflow/workflow_step_action.go` from the module cache (`go env GOMODCACHE`), confirm where `GetWorkflowStatus()` must live (object vs status) before wiring.
- **R8 — SSA field-ownership upgrade.** Objects previously created/updated by the operator (merge) will be re-owned by the new `fieldManager` via `ForceOwnership`. Conflicting external field managers (kubectl, HPA) will be overridden for owned fields — acceptable but must be documented; `CleanupReadLastAppliedAnnotations` handles the legacy annotation.
- **R9 — mutating webhooks + dry-run diff.** The diff variant's SSA dry-run may produce false-positive updates if webhooks mutate objects. *Fallback:* use the simple variant (always-apply) for resources prone to webhook mutation; SSA idempotency prevents update loops.
- **R10 — RSA→ECDSA migration + P-521 caveat.** A `KeyComplexity` change requires a full CA rotation + cluster-wide rolling restart (no mixed-algorithm cluster, §4.4.4). ECDSA is supported by ES/Logstash/Kibana (§4.4.4.1); P-256 is the safe default, P-521 is the least-tested on some JVM/FIPS provider combinations. *Fallback:* keep `rsa-2048` default; validate `ecdsa-p256` end-to-end in envtest/e2e before advertising `ecdsa-p384`/`ecdsa-p521`.
- **R11 — es-handler v9 + disaster37/elasticsearch client drift.** v8.1.5 → v9.0.0 is a breaking change: the client config (`elasticsearch.Config{URL,…}` single URL, no `Transport`/`Addresses`), `Client()` return type, and every API-object type (olivere → `esapi.*`/es-handler local types, §4.5.1). Single-URL constraint drops multi-address failover for external ES. *Fallback:* `go doc github.com/disaster37/es-handler/v9` + `go doc github.com/disaster37/elasticsearch/v9` + the maintainer's own `es9-migration.md` (in the es-handler repo). Verify no use of removed v8 methods (`ClusterGetSettings`/`ClusterPutSettings`/`GetVersion`).

**Implementation-time exact-API commands** (for anything still ambiguous):
```
go mod download github.com/disaster37/operator-sdk-extra/v3
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged/pernode
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/byo
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/helper
go doc github.com/disaster37/operator-sdk-extra/v3/pkg/helper/ssa
```
Consult the module source under `$(go env GOMODCACHE)/github.com/disaster37/operator-sdk-extra/v3@v3.0.7/` for exact `Diff`/`Apply`/`WorkflowStepReconcilerAction` signatures; `$(go env GOMODCACHE)/github.com/disaster37/es-handler/v9@v9.0.0/` and `$(go env GOMODCACHE)/github.com/disaster37/elasticsearch/v9@v9.0.0/` for the new ES client/type surface.

---

## 7. Validation plan (regression)

- `make generate` + `make manifests` — no unintended CRD/spec drift (review `config/crd/bases/*.yaml` and `zz_generated.deepcopy.go`; expect ONLY the additive optional `spec.tls.caRenewalDays` and `spec.tls.keyComplexity` (with its `Enum` validation) (+ equivalent Pki fields if added) and the status-only `.status.tlsWorkflowStatus` on elasticsearch/logstash/kibana/filebeat; NO change to any existing spec field).
- `make test` (envtest) — all suites (`apis/...`, `pkg/...`, `controllers/...`) green. Focus suites: `internal/controller/{elasticsearch,kibana,logstash,filebeat,metricbeat,cerebro,elasticsearchapi,kibanaapi}` + `api/*/v1` webhook suites.
- TLS/cert lifecycle scenario tests (envtest, `TEST=true`) — cover the full ES lifecycle: (1) **bootstrap**: fresh CR → CA + per-node leaves + API cert generated, `IsBootstrapping` set, pods ready; (2) **scale-up**: add a node → new `<node>.crt` appears, existing pods NOT restarted; (3) **scale-down**: remove a node → its keys removed, NO cluster-wide reconcile/rollout; (4) **leaf renew**: force `.../renew-certificates=true` or age the leaf → leaves re-issued + staged rollout; (5) **CA renew/rotate**: age the CA past `caRenewalDays` → saga runs `""→Rotate→Converge→""` with the `ConvergenceCheck` gate; (6) **blackout/expiry recovery**: force all certs expired → `TlsBlackout` condition set + full regeneration + pod recovery; (7) **BYO**: `CertificateSecretRef` set → no child Secret emitted, user secret wired; (8) **trust-model**: after scale-up assert the new node's cert SAN contains its pod name, existing pods keep their old mounted certs, and the cluster reports no TLS handshake/verification errors; (9) **RSA→ECDSA migration**: set `keyComplexity: ecdsa-p256` → `CAContentChanged`/`LeafKeyChanged` triggers `Rotate` (not near expiry), all pods converge on ECDSA certs via the staged rollout, old RSA CA stripped at `Converge`, and no mixed RSA/ECDSA certs observed mid-flight. Also assert the rollout marker changes ONLY on rotation, not on membership change.
- Manual/e2e smoke: create an Elasticsearch CR with `spec.tls.enabled=true` in a kind cluster (`make k8s`), assert Secret `*-ca`/`*-tls` created, pods ready, and certificate-hash annotation triggers rolling restart on forced renewal.

---

## 8. Post-review fix: rollout-marker + API leaf drift

### 8.1 Problem (from code review of the implemented migration)

1. **Rollout-marker invariant broken.** `statefulset_builder.go buildStatefulsets` still prefers `s.Annotations["…/sequence"]`, but nothing sets that annotation anymore (the transport TLS saga `rotation.NewTLSStep` does not write it). The builder therefore **falls back to hashing the whole transport Secret** (`checksum.SHA256sumReader(json.Marshal(s.Data))`), so node add/remove (which add/remove `<node>.crt`/`.key` keys) and every leaf regen change the hash → **cluster-wide StatefulSet rollout**. This violates §4.4.5/§4.4.6's "membership changes must NOT feed the template hash" invariant.
2. **API leaf drift not detected.** `secret_tls_reconciler.go apiLeafNeedsRegeneration` only checks expiry (`crt.NotAfter`), so SAN/CN/subject/key changes on the API cert never re-issue it.

### 8.2 Library API facts (verified @ operator-sdk-extra v3.0.7 module cache)

- `rotation.go` `signalKey()` = `"tls." + s.GetPhaseName().String()` (line 145). The transport step's `phaseName` is `TlsTransportPhase = "TlsTransport"`, so the signal lands at **`data["tls.TlsTransport"]`** (type `*certificate.LayerSignals`), published at line 342 (leaf-only path) and line 455 (`runCASaga` path); `data["rotationRenewed"] = true` on CA saga (line 454).
- `rollout.go`:
  - `type LayerSignals struct { CARotated bool; LeafRegenerated bool; LeafChange *LeafChange; Forced bool }`.
  - `type RolloutPolicy string`; consts `RolloutAlways`, `RolloutOnCAChange`, `RolloutOnAdditive`, `RolloutNever`.
  - `func ShouldRollout(policy RolloutPolicy, sig *LayerSignals) bool` — `RolloutOnAdditive`: `true` when `CARotated` or `LeafChange.Reason ∈ {LeafExpiring, LeafCNChanged, LeafOrgChanged, LeafSubjectChanged, LeafKeyChanged, LeafUsagesChanged, LeafMissing, LeafForceRegen}`, or `LeafSANsChanged/LeafIPsChanged` with non-empty `SANsAdded/IPsAdded`; **`false`** for `LeafNodesChanged` (node add/remove), `LeafNone`, and SAN/IP **removal-only**. `Forced` overrides everything.
  - `func SecretHash(secret *corev1.Secret) (string, error)` (hashes whole `Data`), `func SecretHashAnnotation(secret) (map[string]string, error)`, `func RolloutAnnotation(shouldRollout bool, secret *corev1.Secret, currentHash string) (map[string]string, error)`, const `AnnotationSecretHash = "operator-sdk-extra.webcenter.fr/certificate-hash"`.
- `selfmanaged/backend.go` (exported, usable by the hand-rolled API reconciler): `func CANeedsRenewal(caSecret, spec, now) (bool, error)`, `func CAContentChanged(caSecret, spec) (bool, error)`, `func KeyChanged(spec, cert) (bool, error)`, `func UsagesChanged(spec, cert) (bool, error)`, `func OrganizationsEqual(spec, cert) bool`, `func SubjectRestEqual(a, b pkix.Name) bool`. `TLSSpec` methods: `LeafSubject() pkix.Name`, `Organizations() []string`.

### 8.3 Fix A — transport rollout-marker invariant

**Decision: keep the per-secret pod-template annotation key** (`<elasticsearchcrd.ElasticsearchAnnotationKey>/secret-<name>`, already used), but drive the **transport secret's** value from the library `LayerSignals` + `ShouldRollout(RolloutOnAdditive, …)` instead of hashing the whole Secret. The marker lives **on the StatefulSet pod-template** (owned by the STS step), never on the Secret — so it cannot fight the TLS saga's SSA ownership of the Secret.

**A.1 — `statefulset_builder.go` `buildStatefulsets`** (line 38): add a parameter and special-case the transport secret.

```go
func buildStatefulsets(es *elasticsearchcrd.Elasticsearch, secretsChecksum []*corev1.Secret, configMapsChecksum []*corev1.ConfigMap, isOpenshift bool, transportRolloutMarker string) (statefullsets []*appv1.StatefulSet, err error) {
	// ...
	for _, s := range secretsChecksum {
		var sum string
		switch {
		case s.Name == GetSecretNameForTlsTransport(es) && transportRolloutMarker != "":
			// Transport cert: use the STS-computed rotation marker (changes
			// ONLY on CA/leaf rotation, never on node add/remove).
			sum = transportRolloutMarker
		case s.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)] != "":
			sum = s.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)]
		default:
			j, err := json.Marshal(s.Data)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when convert data of secret %s on json string", s.Name)
			}
			sum, err = checksum.SHA256sumReader(bytes.NewReader(j))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when generate checksum for extra secret %s", s.Name)
			}
		}
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, s.Name)] = sum
	}
	// ...
}
```

**A.2 — `statefulset_reconciler.go` `Read`** (line 64): compute the marker before `buildStatefulsets`.

```go
// After the transport secret is read (around line 153) and before buildStatefulsets:
import (
	// ...
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
)

// ...
transportSecretName := GetSecretNameForTlsTransport(o)
transportMarker := ""
sig, _ := data["tls.TlsTransport"].(*certificate.LayerSignals)
shouldRollout := certificate.ShouldRollout(certificate.RolloutOnAdditive, sig)

markerKey := fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, transportSecretName)
currentMarker := ""
for i := range stsList.Items {
	if v, ok := stsList.Items[i].Spec.Template.Annotations[markerKey]; ok && v != "" {
		currentMarker = v
		break
	}
}

if shouldRollout || currentMarker == "" {
	// Fresh marker: at rotation/bootstrap every node cert + CA changed, so a
	// whole-Secret hash is a correct, stable marker.
	h, err := certificate.SecretHash(transportSecret /* the Secret read at line 145 */)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when hash transport secret")
	}
	transportMarker = h
} else {
	// No rollout: keep the current marker so the pod-template is unchanged.
	transportMarker = currentMarker
}

expectedSts, err := buildStatefulsets(o, secretsChecksum, configMapsChecksum, r.isOpenshift, transportMarker)
```

Signals → marker semantics (what bumps the marker):
- **CA rotation / leaf content change / expiry / forced** → `ShouldRollout==true` → new `SecretHash` → full staged rollout (the existing one-at-a-time gating in `Diff` already limits concurrency).
- **Node added / node removed** → `LeafNodesChanged` (or `LeafNone`) → `ShouldRollout==false` → marker unchanged → **no rollout**; the new pod picks up its `<node>.crt` at `init-filesystem` (copies `${POD_NAME}.crt`/`.key`), the removed node's pods are already gone.
- **SAN/IP removal-only** → `ShouldRollout==false` (RolloutOnAdditive) → no rollout (safe: ES trusts via CA + full SAN set, see §4.4.5.1).

**A.3 — API secret** stays on the whole-Secret hash (single cert; any drift SHOULD roll). Keystore/cacerts/extra secrets stay on the whole-Secret hash. No change.

### 8.4 Fix B — API leaf drift detection

`apiLeafNeedsRegeneration` (line 440) must also detect CN/subject/key/SAN/IP drift, not just expiry.

```go
func apiLeafNeedsRegeneration(sApi *corev1.Secret, spec certificate.TLSSpec) (bool, error) {
	if sApi == nil {
		return true, nil
	}
	raw, ok := sApi.Data["tls.crt"]
	if !ok || len(raw) == 0 {
		return true, nil
	}
	crt, err := certParse(raw)
	if err != nil {
		return false, errors.Wrap(err, "Error when parse API certificate")
	}

	// 1. Expiry.
	if time.Now().After(crt.NotAfter.Add(-(time.Duration(certificate.GetValidRenewalDays(spec)) * 24 * time.Hour))) {
		return true, nil
	}
	// 2. CN + subject drift.
	if crt.Subject.CommonName != spec.CommonName {
		return true, nil
	}
	if !selfmanaged.OrganizationsEqual(spec, crt) {
		return true, nil
	}
	if !selfmanaged.SubjectRestEqual(spec.LeafSubject(), crt.Subject) {
		return true, nil
	}
	// 3. Key algorithm/size drift (e.g. KeyComplexity change).
	if changed, err := selfmanaged.KeyChanged(spec, crt); err != nil {
		return false, err
	} else if changed {
		return true, nil
	}
	// 4. SAN drift (DNS + IP).
	if !stringSetEqual(spec.DNSNames, crt.DNSNames) {
		return true, nil
	}
	if !ipSANsEqual(spec.IPAddresses, crt.IPAddresses) {
		return true, nil
	}
	return false, nil
}

// stringSetEqual / ipSANsEqual: small order-insensitive set comparisons
// (dedupe via certificate.DedupStrings / net.ParseIP). The selfmanaged package
// keeps its equivalents unexported, so add these locally.
```

Rollout is automatic: the API Secret is in `secretsChecksum` and hashed whole, so a re-issued API cert changes the hash → STS rollout. No marker wiring needed for the API layer.

### 8.5 Edge cases

- **First reconcile after upgrade** (saga steady-state, no signal): `sig == nil` → `ShouldRollout == false`; `currentMarker` = the existing pod-template `…/secret-<transport>` value (the old whole-Secret hash) → marker kept → **no rollout on upgrade**.
- **Bootstrap** (no live STS): `currentMarker == ""` → fresh `SecretHash` initializes the marker.
- **TEST=true**: `transportConvergenceCheck` returns `true` immediately and `Diff`'s upgrade gating uses the existing `TEST=="true"` fast paths — marker logic itself is TEST-agnostic.
- **Blackout**: `TlsConditionBlackout` branch in `Diff` forces "reconcile all statefulsets"; the saga's regeneration sets `CARotated/LeafRegenerated` → `ShouldRollout==true` → marker bumps → full rollout. Preserved.
- **BYO** (`CertificateSecretRef`): transport is unaffected (always self-managed); API secret is user-managed and hashed whole (a user's secret change correctly rolls).
- **Multi-nodegroup**: `checksumAnnotations` is computed once and copied per node-group, so one shared `transportMarker` applies everywhere (the transport Secret is cluster-wide).
- **SSA field ownership**: the marker is a pod-template annotation on the StatefulSet (STS-step-owned), never on the Secret — no conflict with the TLS saga's Secret apply.

### 8.6 Tests (envtest, `TEST=true`)

Add/update in `internal/controller/elasticsearch/elasticsearch_controller_test.go` (or a focused suite):
- **scale-up without rollout**: create CR (1 node) → converge; add a node → assert a new `<node>.crt` appears in the transport Secret **and** every StatefulSet's `status.currentRevision == status.updateRevision` (no new revision) and pod-template `…/secret-<transport>` annotation is unchanged.
- **scale-down without rollout**: remove a node → assert the marker annotation is unchanged and no StatefulSet revision bump.
- **CA rotation → full staged rollout**: bump `…/force-regenerate-tls` or age the CA → assert `LayerSignals.CARotated` path bumps the marker and pods roll (one node-group at a time via `Diff` gating).
- **leaf-only renew → rollout**: force `…/force-regenerate-certificates` → assert `LeafRegenerated` bumps the marker.
- **SAN drift → API cert re-issue**: change `spec.tls.selfSignedCertificate.altNames` → assert the API Secret's `tls.crt` SANs change (cert re-issued) and the STS rolls.
- **rollout-absence assertion**: record `sts.Status.CurrentRevision`/`UpdateRevision` (and `Generation`) before and after a node add/remove; assert equal.

### 8.7 Definition of done

```
go build ./...
go vet ./...
go fmt ./...
TEST=true KUBEBUILDER_ASSETS="$(make -s envtest-path)" go test ./internal/controller/elasticsearch/... -run 'ScaleUp|ScaleDown|CARotation|LeafRenew|SANDrift' -count 1
make test   # full envtest suite green
```

Manual assertion: after scale-up/scale-down, `kubectl get statefulset -o jsonpath='{.status.currentRevision}{" "}{.status.updateRevision}'` is identical for every node-group, and the pod-template annotation `…/secret-<transport-secret-name>` is byte-identical to its pre-change value.
