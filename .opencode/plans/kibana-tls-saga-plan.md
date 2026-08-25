# Kibana TLS: adopt the operator-sdk-extra/v3 certificate rotation saga

## 1. Overview & goals

Replace the Kibana controller's hand-written, goca-based TLS/certificate logic
(`internal/controller/kibana/secret_tls_reconciler.go` + `secret_tls_builder.go`)
with the reusable TLS rotation saga from `github.com/disaster37/operator-sdk-extra/v3`
(the library is already a direct dependency at **v3.0.7**, see `go.mod` line 17).

Goals:

1. **CA renewal** — full `"" → Rotate → Converge → ""` saga (emit new CA + leaf with
   `ca.crt = newCA||oldCA` bundle → wait for the Kibana Deployment to roll → strip the
   old CA).
2. **Leaf renewal** — leaf-only re-sign against the existing CA (no CA rotation).
3. **Force-renew annotations (LOCAL)** — two Kibana-specific annotations:
   - `kibana.k8s.webcenter.fr/force-renew-tls` → full CA + leaf rotation.
   - `kibana.k8s.webcenter.fr/force-renew-certificates` → leaf-only renewal.
4. **Blackout** — **out of scope**. Decision: no blackout for Kibana. The saga has no
   "pause renewals"/blackout mechanism, and the repo's `TlsBlackout` is Elasticsearch-
   specific (hard-expired transport certs → delete-all-pods recovery). Kibana is a single
   `Deployment` whose TLS consumers skip CA verification (`kibanaapi/helper.go` sets
   `InsecureSkipVerify: true`) and whose expired certs the saga already auto-renews; the
   existing secret-hash pod-template annotation rolls the Deployment.

Reference implementation already in-tree: the Elasticsearch transport layer
(`internal/controller/elasticsearch/secret_tls_reconciler.go`, `newTlsTransportReconciler`)
already uses `rotation.NewTLSStep`. This plan mirrors that pattern for Kibana's single-leaf
case (closer to the ES API layer's `selfmanaged` usage).

## 2. Upstream saga surface (verified @ v3.0.7, branch `v3`, commit `0ab68906`)

Import paths and symbols used:

| Import | Symbols |
|---|---|
| `github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared` | `ConditionName`, `PhaseName` |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/apis/workflow` | `WorkflowStatus`, `WorkflowPhase` |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate` | `TLSSpec`, `CertificateSubject`, `TLSSpecProviderFunc`, `CertificateCustomizer`, `CertificateCustomizerFunc`, `LayerSignals`, `AnnotationForceRegenerateAll`, `AnnotationForceRegenerateLeaf`, `KeyAlgorithmECDSA/RSA`, `CurveP256/P384/P521` |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation` | `NewTLSStep`, `WithConvergenceCheck`, `WithLabelsDecorator`, `WithAnnotationsDecorator`, `WithForceRegenerateAllAnnotation`, `WithForceRegenerateLeafAnnotation`, `WithCertificateCustomizer`, `ConvergenceCheck`, `PhaseRotate`, `PhaseConverge` |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged` | `NewSelfManagedBackend`, `CASecretSuffix` (`"-ca"`), `CAKey` (`ca.crt`), `CertKey` (`tls.crt`), `KeyKey` (`tls.key`), `CAKeyPrivate` (`ca.key`), `CRLKey` (`ca.crl`) |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow` | `WorkflowStepReconcilerActionWithDiff` |
| `github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase` | `MultiPhaseStepReconcilerAction`, `MultiPhaseRead` |

Saga mechanics (from `rotation/rotation.go`):

- `NewTLSStep(c, phaseName, conditionName, recorder, fieldManager, backend, provider, opts...)`
  returns a `workflow.WorkflowStepReconcilerActionWithDiff[T, client.Object]`.
- Phases (persisted in `status.tlsWorkflowStatus.currentPhase`, requires the object to
  implement `WorkflowStatusGetter`): `"" → Rotate → Converge → ""`.
- At phase `""`: reads force annotations; computes `caNeed` (`!caExists || forceAll ||
  CANeedsRenewal || CAContentChanged`); computes `LeafChange` via `LeafManager.LeafNeedsChange`.
  `caNeed` → `runCASaga` (new CA + leaf, bundle old CA into leaf `ca.crt`); else non-zero
  `LeafChange` → `DesiredLeafWithCA` (leaf-only, no phase write); else steady.
- `Rotate`: stable (expected == current), gated by `ConvergenceCheck`; on converge →
  `AdvancePhase(Converge)` in `OnDiff`.
- `Converge`: strip old CA from leaf `ca.crt`; on success → `AdvancePhase("")`.
- Force annotations are honored only at `""`, read as `== "true"`, force-all wins, and the
  honored annotation is removed via `client.Update` in `OnSuccess` after a successful Apply.
- The selfmanaged backend emits two Secrets: `<secretName>-ca` (`ca.crt` + `ca.key` [+ `ca.crl`
  if `GenerateCRL`]) and `<secretName>` (`tls.crt` + `tls.key` + `ca.crt`, type
  `kubernetes.io/tls`). It does **not** set owner references or TypeMeta — decorators must set
  TypeMeta (SSA requires `apiVersion`/`kind`).

## 3. Design decisions (resolved)

- **Backend**: `selfmanaged.NewSelfManagedBackend[*kibanacrd.Kibana]()` (single CA + single leaf).
- **BYO / TLS-disabled**: a thin wrapper step short-circuits `Read` (emits nothing) when
  `!o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls()`; the saga only runs
  for self-managed TLS. (The saga can't switch backend per-reconcile; `RequiresRotationSaga()` is
  static, so BYO must be short-circuited before the saga's phase logic runs.)
- **Local annotation keys**:
  - `AnnotationForceRenewTLS = "kibana.k8s.webcenter.fr/force-renew-tls"` (CA + leaf) →
    wired via `rotation.WithForceRegenerateAllAnnotation`.
  - `AnnotationForceRenewCertificates = "kibana.k8s.webcenter.fr/force-renew-certificates"`
    (leaf only) → wired via `rotation.WithForceRegenerateLeafAnnotation`.
- **CA secret naming**: align to the saga's hardcoded `<leafName>-ca` suffix. Leaf name is
  `GetSecretNameForTls` (`<name>-tls-kb`), so the CA secret becomes **`<name>-tls-kb-ca`**
  (today's PKI secret is `<name>-pki-kb`). `GetSecretNameForPki` is changed to return the new
  name, and a one-time legacy cleanup deletes `<name>-pki-kb` once the new CA exists
  (mirrors ES `cleanupLegacyTransportCASecret`).
- **`caRenewalDays`**: the saga's `CANeedsRenewal` uses the shared `RenewalDays` window; a
  `CertificateCustomizer` re-checks the CA against `caRenewalDays` and sets the force-all
  annotation (mirrors ES `transportCARenewalCustomizer`).
- **Key algorithm**: `KeyComplexity` maps to `TLSSpec.KeyAlgorithm`/`Curve`/`KeySize`;
  empty → legacy `KeySize` (RSA, default 2048), mirroring ES `applyKeyComplexity`.
- **`ca.crl` / `ca.pub`**: dropped (no consumer after goca removal). Optional: set
  `TLSSpec.GenerateCRL = true` to preserve `ca.crl`; recommended `false` (matches ES API layer).

## 4. Files to change

### 4.1 Create / rewrite

- **`internal/controller/kibana/secret_tls_reconciler.go`** (rewrite) — saga step + wrapper +
  spec provider + customizer + convergence check + legacy cleanup + local annotation constants.

### 4.2 Modify

- **`api/kibana/v1/kibana_types.go`** — add `TlsWorkflowStatus workflow.WorkflowStatus` to
  `KibanaStatus`; add `workflow` import.
- **`api/kibana/v1/kibana_func.go`** — add `GetWorkflowStatus()` on `*KibanaStatus`; add `workflow` import.
- **`api/kibana/v1/zz_generated.deepcopy.go`** — regenerate via `make generate`.
- **`internal/controller/kibana/helper.go`** — change `GetSecretNameForPki` return value to
  `"<name>-tls-kb-ca"` + update comment.
- **`internal/controller/kibana/kibana_controller.go`** — replace the TLS step entry in
  `stepReconcilers` with the new saga step (drop the `*corev1.Secret`/`NewObjectMultiPhaseStepReconcilerAction`
  wrapper; pass `logger`).
- **`internal/controller/kibana/helper_test.go`** — update `TestGetSecretNameForPki` expected value.
- **`internal/controller/kibana/kibana_controller_test.go`** — update PKI/CA-secret assertions
  (owner refs no longer present; assert `ca.crt`/`ca.key` keys); add CA-rotation + leaf-renew
  tests (mirror ES `TestElasticsearchControllerCARotation`/`LeafRenew`).

### 4.3 Delete

- **`internal/controller/kibana/secret_tls_builder.go`** — delete (goca builders, now dead).
- **`internal/controller/kibana/secret_tls_builder_test.go`** — delete.

## 5. Data structures & constants

### 5.1 Annotation constants (in `secret_tls_reconciler.go`)

```go
const (
    // AnnotationForceRenewTLS forces a full CA + leaf rotation (read as "true").
    AnnotationForceRenewTLS = "kibana.k8s.webcenter.fr/force-renew-tls"
    // AnnotationForceRenewCertificates forces leaf-only renewal (read as "true").
    AnnotationForceRenewCertificates = "kibana.k8s.webcenter.fr/force-renew-certificates"
)
```

### 5.2 Status field (`kibana_types.go`)

```go
// TlsWorkflowStatus tracks the CA rotation saga phase.
// +optional
TlsWorkflowStatus workflow.WorkflowStatus `json:"tlsWorkflowStatus,omitempty"`
```

```go
// kibana_func.go
// GetWorkflowStatus implement the workflow.WorkflowStatusGetter
func (s *KibanaStatus) GetWorkflowStatus() *workflow.WorkflowStatus {
    return &s.TlsWorkflowStatus
}
```

(No other spec/status/condition changes. `TlsCondition "TlsReady"` / `TlsPhase "Tls"` are kept.)

## 6. Function signatures (new code in `secret_tls_reconciler.go`)

```go
// Wrapper step: saga for self-managed TLS; no-op for BYO / TLS-disabled.
type tlsReconciler struct {
    workflow.WorkflowStepReconcilerActionWithDiff[*kibanacrd.Kibana, client.Object]
}

func newTlsReconciler(c client.Client, recorder record.EventRecorder, log *logrus.Entry) multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, client.Object]

// Read short-circuits the saga when TLS is disabled or user-managed (BYO);
// otherwise performs legacy-CA cleanup and delegates to the saga.
func (r *tlsReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (multiphase.MultiPhaseRead[client.Object], reconcile.Result, error)

// kibanaTLSSpec builds the computed TLSSpec for a Kibana CR.
func kibanaTLSSpec(o *kibanacrd.Kibana) certificate.TLSSpec

// applyKeyComplexity maps the CR's KeyComplexity/legacy KeySize to TLSSpec
// key algorithm/curve/size (mirror of elasticsearch.applyKeyComplexity).
func applyKeyComplexity(spec *certificate.TLSSpec, complexity string, legacyKeySize *int)

// kibanaCARenewalCustomizer forces a CA rotation when the current CA is within
// caRenewalDays of expiry (sets AnnotationForceRenewTLS on o).
func kibanaCARenewalCustomizer(c client.Client, log *logrus.Entry) certificate.CertificateCustomizer[*kibanacrd.Kibana]

// kibanaConvergenceCheck gates the Rotate→Converge transition on the Kibana
// Deployment having rolled to the latest generation.
func kibanaConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*kibanacrd.Kibana]

// cleanupLegacyKibanaCASecret deletes the pre-saga PKI secret <name>-pki-kb once
// the new saga CA secret <name>-tls-kb-ca exists. Best-effort.
func cleanupLegacyKibanaCASecret(ctx context.Context, c client.Client, o *kibanacrd.Kibana, logger *logrus.Entry) error

// certParse parses the first CERTIFICATE PEM block (Kibana-local copy of the ES helper).
func certParse(pemBytes []byte) (*x509.Certificate, error)
```

## 7. Reconcile flow integration (pseudocode)

`NewKibanaReconciler` wires the step (unchanged position — TLS runs before CA-Elasticsearch,
credential, configmap, service, deployment):

```go
stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, client.Object]{
    ...
    newTlsReconciler(c, recorder, logger),   // was: NewObjectMultiPhaseStepReconcilerAction[...](newTlsReconciler(c, recorder))
    ...
}
```

`newTlsReconciler`:

```go
saga := rotation.NewTLSStep[*kibanacrd.Kibana](
    c, TlsPhase, TlsCondition, recorder, common.FieldManager,
    selfmanaged.NewSelfManagedBackend[*kibanacrd.Kibana](),
    certificate.TLSSpecProviderFunc[*kibanacrd.Kibana](kibanaTLSSpec),
    rotation.WithConvergenceCheck(kibanaConvergenceCheck(c)),
    rotation.WithCertificateCustomizer(kibanaCARenewalCustomizer(c, log)),
    rotation.WithForceRegenerateAllAnnotation(AnnotationForceRenewTLS),
    rotation.WithForceRegenerateLeafAnnotation(AnnotationForceRenewCertificates),
    rotation.WithLabelsDecorator(func(o *kibanacrd.Kibana, obj client.Object) {
        obj.SetLabels(getLabels(o))
    }),
    rotation.WithAnnotationsDecorator(func(o *kibanacrd.Kibana, obj client.Object) {
        obj.SetAnnotations(getAnnotations(o))
        obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
    }),
)
return &tlsReconciler{WorkflowStepReconcilerActionWithDiff: saga}
```

`tlsReconciler.Read`:

```go
if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
    return multiphase.NewMultiPhaseRead[client.Object](), reconcile.Result{}, nil // BYO / disabled
}
if err := cleanupLegacyKibanaCASecret(ctx, r.Client(), o, logger); err != nil {
    logger.Warnf("Failed to clean up legacy Kibana CA secret: %s", err.Error())
}
return r.WorkflowStepReconcilerActionWithDiff.Read(ctx, o, data, logger)
```

Saga `Read` behavior for self-managed Kibana (library code, phase `""`):

1. `computeSpec` = provider + customizer (customizer may set `AnnotationForceRenewTLS` when the
   CA is within `caRenewalDays`).
2. Load current leaf `<name>-tls-kb` and CA `<name>-tls-kb-ca` (NotFound → nil).
3. Publish sanitized `data["tlsSecret"]`/`data["caSecret"]` (+ parsed `leafCert`/`caCert`).
4. Read force flags. `caNeed = !caExists || forceAll || CANeedsRenewal || CAContentChanged`.
   `leafChg = LeafNeedsChange(leaf, spec)` (Missing/Expiring/CN/SAN/key/subject drift).
5. `caNeed` → full CA saga (new CA + leaf, bundle old CA into leaf `ca.crt`,
   `data["rotationRenewed"]=true`, `LayerSignals{CARotated,LeafRegenerated,Forced}` under
   `data["tls.Tls"]`). Else `leafChg != 0` → `DesiredLeafWithCA` (leaf-only,
   `LayerSignals{LeafRegenerated,Forced}`). Else steady.
6. `OnSuccess`: advance `""→Rotate` if `rotationRenewed`; remove honored force annotation.
7. `OnDiff` (Rotate): `kibanaConvergenceCheck` → when converged `Rotate→Converge`; not converged →
   `RequeueAfter: 10s` (library default).
8. `Converge`: leaf `ca.crt` = CA `ca.crt` (strip bundle); `OnSuccess`: `Converge→""`.

Deployment roll: the existing `deployment_reconciler.go` already hashes `GetSecretNameForTls`
into the pod template (no change). The leaf Secret data changes at `""` (new leaf/bundle) and
`Converge` (strip) → checksum changes → Deployment rolls. No `LayerSignals` consumption is
needed for Kibana (single leaf secret, no membership-vs-rotation ambiguity).

## 8. Edge cases & error handling

- **First run (saga not initialized)**: phase `""`, CA + leaf missing → `caNeed` → full CA
  saga; `OnSuccess` advances to `Rotate`; convergence check gates; `Converge` strips (no old
  CA to strip → leaf already single CA); `""` steady. `TlsWorkflowStatus` must be persisted
  (hence the `WorkflowStatusGetter` addition) or the saga loops.
- **CA expired vs leaf expired**: CA expiry → `CANeedsRenewal` → full saga. Leaf expiry only →
  `LeafNeedsChange(LeafExpiring)` → leaf-only re-sign (CA untouched).
- **Blackout/expiry recovery**: N/A for Kibana (expired certs are normal-renewed by the saga;
  Deployment rolls). No `TlsBlackout` condition.
- **Force annotation malformed**: only `== "true"` triggers; any other value is ignored
  (library `forceFlags`). Non-saga backends would warn+remove, but Kibana's BYO path is
  short-circuited before that, so a stray force annotation on a BYO Kibana is simply ignored.
- **Force-all + force-leaf both set**: force-all wins (library behavior); both are removed on success.
- **Force annotation on failed Apply**: left in place, retried next cycle (library behavior).
- **Secret missing/corrupt**: missing → treated as "generate". Corrupt `ca.crt`/`tls.crt` →
  `CANeedsRenewal`/`LeafNeedsChange` return error → reconcile error → exponential backoff
  (rate limiter). The customizer returns `(base, nil)` on unparseable CA (avoid breaking bootstrap).
- **Owner references**: saga-managed Secrets have **no** owner reference (consistent with ES
  transport). Consequence: `Owns(&corev1.Secret{})` won't watch them, and they are not GC'd by
  the API server on CR delete. Deletion relies on the multiphase reconciler's cleanup path
  (same as ES transport). Flag for verification in §11.
- **Watch/requeue triggers**: phase advances write CR status → reconcile re-queued. Convergence
  gate requeues at 10s. Customizer sets an annotation → the CR Update it performs (via the
  saga's `removeForceAnnotations` / status update) triggers the next reconcile.
- **Idempotency**: steady state registers current==expected → SSA apply no-op; label/annotation
  drift is re-applied by the decorators every cycle.

## 9. Logging & event conventions

- Follow the existing style: `logger.Debugf/Infof/Warnf`, `emperror.dev/errors.Wrapf(err, "...")`
  with capitalized messages ("Error when read existing secret %s", etc.).
- `r.Recorder().Event(o, corev1.EventTypeWarning/Normal, reason, message)` where applicable
  (library already emits `TLSForceUnsupported` for non-saga backends; not hit by Kibana's
  self-managed path). No new events are required beyond the library's defaults.

## 10. Testing plan

Framework: `testify/suite` + `envtest` (existing convention; see `suite_test.go`).

### 10.1 Unit tests (fast, no envtest)

- **`helper_test.go`** — update `TestGetSecretNameForPki` → expect `"test-tls-kb-ca"`.
- **`secret_tls_reconciler_test.go`** (new) — table tests for:
  - `kibanaTLSSpec` (defaults: SecretName/CN/CA CN `"<name>-api"`/O/OU/subject/DNS names/validity/
    renewal; `KeyComplexity` mapping; `AltNames`/`AltIps` SANs).
  - `applyKeyComplexity` (rsa-2048/4096, ecdsa-p256/p384/p521, empty→legacy KeySize, invalid→RSA 2048).
  - `certParse` (valid PEM, empty, garbage).
  - `kibanaCARenewalCustomizer` with a fake client: nil `caRenewalDays` → unchanged; CA missing →
    unchanged; CA outside window → unchanged; CA within window → sets `AnnotationForceRenewTLS`.

### 10.2 envtest integration (extend `kibana_controller_test.go`, mirror ES suite)

- **Bootstrap** (existing `TestKibanaController`): update PKI secret assertions — replace
  `assert.NotEmpty(s.OwnerReferences)` with `assert.NotEmpty(s.Data["ca.crt"])` /
  `assert.NotEmpty(s.Data["ca.key"])`; the leaf secret assertion (already checks Data) stays.
  Assert `status.tlsWorkflowStatus.currentPhase` converges to `""`.
- **CA rotation** (new `TestKibanaControllerCARotation`): set
  `Annotations[AnnotationForceRenewTLS]="true"`, wait for observedGeneration, assert the CA
  secret `ca.crt` changed and the annotation was cleared.
- **Leaf renew** (new `TestKibanaControllerLeafRenew`): set
  `Annotations[AnnotationForceRenewCertificates]="true"`, assert leaf `tls.crt` changed while CA
  `ca.crt` unchanged, annotation cleared.
- **BYO** (new): set `Spec.Tls.CertificateSecretRef` → assert no `<name>-tls-kb` /
  `<name>-tls-kb-ca` secrets created and no phase writes.

### 10.3 Manual verification

```
make generate manifests      # deepcopy + CRD (status.tlsWorkflowStatus)
make vet
make build                   # generate fmt vet + go build
make test                    # envtest (KUBEBUILDER_ASSETS + ES_OPERATOR_ENVTEST=true)
```

Manual smoke (optional): `make run`, create a Kibana CR, watch the secrets; apply each force
annotation and confirm rotation/renewal and annotation cleanup.

## 11. Risks & open questions

- **Owner references / GC**: saga Secrets lack owner refs. Confirm the multiphase Delete/cleanup
  removes `<name>-tls-kb` and `<name>-tls-kb-ca` on CR deletion (ES transport already relies on
  this). If cleanup relies on owner refs, add a `WithAnnotationsDecorator`-adjacent owner-ref
  decorator (or delete in the controller's `Delete` hook).
- **`ca.crl`/`ca.pub` dropped**: no consumer after goca removal. If any downstream expects
  `ca.crl`, set `TLSSpec.GenerateCRL = true`.
- **`caRenewalDays` mapping**: implemented via customizer (exact), not via the saga's shared
  `RenewalDays`. This mirrors ES transport but is operator-side glue; a future library
  `CARenewalDays` field would simplify it.
- **`pkg/pki`**: `pki.NeedRenewCertificate` remains used by the ES transport customizer and
  (after this change) the Kibana customizer. `pki.LoadRootCA`/`goca` usage in Kibana is removed;
  `goca` remains a dependency while logstash/filebeat still use it.
- **Legacy PKI secret cleanup** is best-effort (logged, non-fatal). The old `<name>-pki-kb`
  secret is left if the new CA never materializes.

## 12. Ordered task list

1. `api/kibana/v1/kibana_types.go`: add `TlsWorkflowStatus` field + `workflow` import.
2. `api/kibana/v1/kibana_func.go`: add `GetWorkflowStatus()`.
3. `internal/controller/kibana/helper.go`: change `GetSecretNameForPki` → `<name>-tls-kb-ca`.
4. `internal/controller/kibana/secret_tls_reconciler.go`: rewrite per §6/§7 (saga + wrapper +
   provider + customizer + convergence + cleanup + annotation constants).
5. Delete `internal/controller/kibana/secret_tls_builder.go` + `secret_tls_builder_test.go`.
6. `internal/controller/kibana/kibana_controller.go`: swap the TLS step entry (pass `logger`).
7. `make generate` (regenerate `zz_generated.deepcopy.go`).
8. Update `helper_test.go` + `kibana_controller_test.go`; add new tests (§10).
9. `make manifests` (CRD); `make vet`; `make build`; `make test`.
