# Filebeat & Logstash: adopt the operator-sdk-extra/v3 certificate rotation saga

## 1. Goal, scope, and non-goals

Replace the hand-rolled, goca-based TLS certificate generation/renewal in the
**Filebeat** and **Logstash** controllers with the reusable certificate rotation
saga from `github.com/disaster37/operator-sdk-extra/v3` (already used by
Elasticsearch and Kibana in this repo — those are the reference implementation and
**must not be touched**).

**Metricbeat is explicitly OUT OF SCOPE.** Research confirmed Metricbeat has no
internal certificate generation: its `MetricbeatSpec` has no `Pki` field, and its
controller has no `secret_tls_reconciler.go` / `secret_tls_builder.go`. It only
reads the Elasticsearch API CA (`secret_ca_elasticsearch_reconciler.go`). There is
nothing to migrate. (Confirmed with the user.)

This plan mirrors the in-tree Kibana migration (single-leaf saga) but uses the
library's **`selfmanaged/pernode`** backend because Filebeat/Logstash manage
**multiple** certificates (one per `spec.pki.tls` entry) in a single opaque Secret.

## 2. Current state (researched)

### 2.1 Reference implementations already migrated (do NOT touch)

- `internal/controller/kibana/secret_tls_reconciler.go` — `rotation.NewTLSStep`
  + `selfmanaged.NewSelfManagedBackend`, wrapper `tlsReconciler` with `Read`
  short-circuit, `kibanaTLSSpec`, `applyKeyComplexity`, `kibanaCARenewalCustomizer`,
  `kibanaConvergenceCheck`, `cleanupLegacyKibanaCASecret`, `certParse`, and local
  force-renew annotation constants.
- `internal/controller/elasticsearch/secret_tls_reconciler.go` — same saga +
  `selfmanaged/pernode` for transport (multi-cert), plus an API-layer single-leaf
  step. `transportNodeSpecProvider` implements `pernode.NodeSpecProvider`.

### 2.2 Files being replaced/removed (hand-rolled goca logic)

- `internal/controller/filebeat/secret_tls_reconciler.go` — `Read`/`Diff` build
  a goca CA (`<name>-pki-fb`) + an opaque leaf Secret (`<name>-tls-fb`) holding
  `ca.crt` and one `<entry>.crt`/`<entry>.key` per key in `spec.pki.tls`.
- `internal/controller/filebeat/secret_tls_builder.go` — `buildPkiSecret`,
  `buildTlsSecret`, `generateCertificate`, `updateSecret`, `getLabelsForTlsSecret`.
- `internal/controller/logstash/secret_tls_reconciler.go` + `secret_tls_builder.go`
  — identical pattern with `<name>-pki-ls` / `<name>-tls-ls` and
  `LogstashTlsSpec` (embeds `TlsSelfSignedCertificateSpec` + `consumer`).

The old leaf Secret data shape is exactly what the saga's `pernode` backend emits:
`ca.crt` + `<node>.crt` / `<node>.key` per node — so consumers that mount
`/usr/share/filebeat/certs/` (Filebeat) or `/usr/share/logstash/...` (Logstash)
keep working with **no pod-spec change**.

### 2.3 Library surface (verified against `v3` branch, module v3.0.7)

Import paths and symbols used:

| Import | Symbols |
|---|---|
| `.../pkg/apis/shared` | `ConditionName`, `PhaseName` |
| `.../pkg/apis/workflow` | `WorkflowStatus` |
| `.../pkg/controller/certificate` | `TLSSpec`, `CertificateSubject`, `TLSSpecProviderFunc`, `KeyAlgorithmRSA`, `KeyAlgorithmECDSA`, `CurveP256/P384/P521` |
| `.../pkg/controller/certificate/rotation` | `NewTLSStep`, `WithConvergenceCheck`, `WithLabelsDecorator`, `WithAnnotationsDecorator`, `WithForceRegenerateAllAnnotation`, `WithForceRegenerateLeafAnnotation`, `ConvergenceCheck` |
| `.../pkg/controller/certificate/selfmanaged` | `CASecretSuffix` (`"-ca"`), `CAKey` (`ca.crt`), `CAKeyPrivate` (`ca.key`) |
| `.../pkg/controller/certificate/selfmanaged/pernode` | `NewPerNodeBackend`, `NodeSpecProvider` |
| `.../pkg/controller/workflow` | `WorkflowStepReconcilerActionWithDiff` |
| `.../pkg/controller/multiphase` | `MultiPhaseStepReconcilerAction`, `MultiPhaseRead` |

Key facts:

- `rotation.NewTLSStep(c, phaseName, conditionName, recorder, fieldManager,
  backend, provider, opts...)` returns a
  `workflow.WorkflowStepReconcilerActionWithDiff[T, client.Object]`.
- Saga phases (persisted in `status.tlsWorkflowStatus.currentPhase` via a
  `WorkflowStatusGetter`): `"" → Rotate → Converge → ""`.
  - `""`: compute `caNeed` (missing CA / force-all / `CANeedsRenewal` /
    `CAContentChanged`); compute leaf drift via `LeafManager.LeafNeedsChange`.
    `caNeed` → `runCASaga` (new CA + leaf, bundle old CA into leaf `ca.crt`);
    else leaf drift → `DesiredLeafWithCA` (leaf-only, no phase write); else steady.
  - `Rotate`: stable (expected == current), gated by the injected
    `ConvergenceCheck`; on converge → advance to `Converge`.
  - `Converge`: strip old CA from leaf `ca.crt`; on success → advance to `""`.
- The `selfmanaged`/`pernode` backends emit **two** Secrets: a CA Secret
  (`<leafName>-ca`, keys `ca.crt` + `ca.key`) and a leaf Secret (`<leafName>`).
  The `pernode` leaf Secret is `SecretTypeOpaque` with `ca.crt` +
  `<node>.crt`/`<node>.key`. **Neither Secret carries an owner reference.**
- `pernode.NodeSpecProvider[T]`:
  ```go
  type NodeSpecProvider[T object.MultiPhaseObject] interface {
      ExpectedNodeNames(o T) ([]string, error)
      NodeCertSpec(o T, nodeName string) (cn string, dnsNames []string, ips []string, err error)
  }
  ```
  `pernode.NewPerNodeBackend[T](provider)`. The base `TLSSpec` supplies the shared
  subject (O/OU/C/L/P), validity, renewal window, and key algorithm/size; per-node
  `NodeCertSpec` overrides CN/DNS/IPs only.
- `TLSSpec` **defaults to ECDSA** (`EffectiveKeyAlgorithm() == KeyAlgorithmECDSA`).
  To preserve today's RSA-2048 goca behavior, `KeyAlgorithm` **must** be set to
  `KeyAlgorithmRSA` and `KeySize` mapped from `spec.pki.keySize`.

## 3. Design decisions (resolved)

- **Backend:** `pernode.NewPerNodeBackend` for both components; "node" == a key of
  `spec.pki.tls`. (Single-leaf `selfmanaged` cannot express N certificates.)
- **Key algorithm preserved:** `KeyAlgorithm = RSA`, `KeySize = spec.pki.keySize`
  (default 2048). Avoids an ECDSA switch (which would force an unnecessary CA
  rotation) and avoids `ValidateContent` rejecting `KeySize=2048` under ECDSA.
- **CA secret naming:** saga hardcodes `<leafName>-ca`. Leaf name stays
  `GetSecretNameForTls` (`<name>-tls-fb` / `<name>-tls-ls`), so the CA Secret
  becomes `<name>-tls-fb-ca` / `<name>-tls-ls-ca`. `GetSecretNameForPki` is changed
  to return the new name; a one-time best-effort cleanup deletes the legacy
  `<name>-pki-fb` / `<name>-pki-ls` once the new CA exists.
- **Force-renew annotations (per component):** added for parity with Kibana/ES,
  wired via `WithForceRegenerateAllAnnotation` / `WithForceRegenerateLeafAnnotation`.
- **No CA customizer:** Filebeat/Logstash `PkiSpec` has no `caRenewalDays` or
  `keyComplexity`, so there is nothing to map; the saga's shared `RenewalDays`
  window drives both CA and leaf renewal (same as today's single window).
- **Convergence check:** wait for the component's single StatefulSet to roll
  (`ObservedGeneration == Generation`, `CurrentReplicas == UpdatedReplicas ==
  Replicas`); fast-forward under envtest.
- **Rollout:** unchanged — the StatefulSet builder already hashes the whole leaf
  Secret data into `…/secret-<name>` pod-template annotations, so any leaf/CA
  change rolls the StatefulSet. No `LayerSignals` consumption is required.
- **Leaf `Organization` (O) field changes:** today's goca leaf certs set
  `O = <entry name>`; the saga's shared subject sets `O = <CR name>` (matching
  ES/Kibana convention). This is a one-time, non-breaking change: trust is
  CA-anchored, not O-anchored. Documented under §9 risks.
- **goca removed:** after migration `pkg/pki.LoadRootCA` (the only remaining goca
  consumer) is dead; remove it and its test and drop `github.com/disaster37/goca`
  via `go mod tidy`. `pkg/pki.NeedRenewCertificate` stays (used by ES + Kibana).

## 4. Files to create, modify, delete

### 4.1 Create / rewrite

- `internal/controller/filebeat/secret_tls_reconciler.go` — **rewrite** (saga +
  pernode provider + wrapper + convergence + legacy cleanup).
- `internal/controller/logstash/secret_tls_reconciler.go` — **rewrite** (same).
- `internal/controller/filebeat/secret_tls_reconciler_test.go` — **new** unit tests.
- `internal/controller/logstash/secret_tls_reconciler_test.go` — **new** unit tests.

### 4.2 Modify

- `api/beat/v1/filebeat_types.go` — add `TlsWorkflowStatus` to `FilebeatStatus` + `workflow` import.
- `api/beat/v1/filebeat_func.go` — add `GetWorkflowStatus()` on `*FilebeatStatus` + `workflow` import.
- `api/logstash/v1/logstash_types.go` — add `TlsWorkflowStatus` to `LogstashStatus` + `workflow` import.
- `api/logstash/v1/logstash_func.go` — add `GetWorkflowStatus()` on `*LogstashStatus` + `workflow` import.
- `internal/controller/filebeat/helper.go` — `GetSecretNameForPki` → `<name>-tls-fb-ca` + comment.
- `internal/controller/logstash/helper.go` — `GetSecretNameForPki` → `<name>-tls-ls-ca` + comment.
- `internal/controller/filebeat/filebeat_controller.go` — swap TLS step entry (pass `logger`).
- `internal/controller/logstash/logstash_controller.go` — swap TLS step entry (pass `logger`).
- `pkg/pki/common.go` — remove `LoadRootCA` + `goca` import (keep `NeedRenewCertificate`).
- `pkg/pki/common_test.go` — remove `LoadRootCA` test + `goca` import.
- `go.mod` / `go.sum` — `go mod tidy` drops `github.com/disaster37/goca`.
- Regenerate: `api/beat/v1/zz_generated.deepcopy.go`,
  `api/logstash/v1/zz_generated.deepcopy.go`,
  `config/crd/bases/beat.k8s.webcenter.fr_filebeats.yaml`,
  `config/crd/bases/logstash.k8s.webcenter.fr_logstashes.yaml` (via `make generate manifests`).

### 4.3 Delete

- `internal/controller/filebeat/secret_tls_builder.go`
- `internal/controller/filebeat/secret_tls_builder_test.go`
- `internal/controller/logstash/secret_tls_builder.go`
- `internal/controller/logstash/secret_tls_builder_test.go`

### 4.4 Tests to update

- `internal/controller/filebeat/helper_test.go` — `TestGetSecretNameForPki` → `"test-tls-fb-ca"`.
- `internal/controller/logstash/helper_test.go` — `TestGetSecretNameForPki` → `"test-tls-ls-ca"`.
- `internal/controller/filebeat/filebeat_controller_test.go` — replace
  `assert.NotEmpty(s.OwnerReferences)` on the PKI/TLS Secrets with Data-key
  assertions (§8.2).
- `internal/controller/logstash/logstash_controller_test.go` — same.

## 5. Data structures & signatures (new code)

### 5.1 `api/beat/v1/filebeat_types.go`

```go
// FilebeatStatus gains:
// TlsWorkflowStatus tracks the CA rotation saga phase.
// +optional
TlsWorkflowStatus workflow.WorkflowStatus `json:"tlsWorkflowStatus,omitempty"`
```
Add import `"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/workflow"`.

### 5.2 `api/beat/v1/filebeat_func.go`

```go
// GetWorkflowStatus implement the workflow.WorkflowStatusGetter interface
func (s *FilebeatStatus) GetWorkflowStatus() *workflow.WorkflowStatus {
    return &s.TlsWorkflowStatus
}
```

(Identical additions for `LogstashStatus` in `api/logstash/v1/`.)

### 5.3 `internal/controller/filebeat/secret_tls_reconciler.go` (full rewrite)

```go
package filebeat

const (
    TlsCondition shared.ConditionName = "TlsReady"
    TlsPhase     shared.PhaseName     = "Tls"

    // AnnotationForceRenewTLS forces a full CA + leaf rotation (read as "true").
    AnnotationForceRenewTLS = "filebeat.k8s.webcenter.fr/force-renew-tls"
    // AnnotationForceRenewCertificates forces leaf-only renewal (read as "true").
    AnnotationForceRenewCertificates = "filebeat.k8s.webcenter.fr/force-renew-certificates"
)

// tlsReconciler wraps the TLS rotation saga for Filebeat self-managed PKI.
// PKI-disabled / empty-tls paths are short-circuited in Read.
type tlsReconciler struct {
    workflow.WorkflowStepReconcilerActionWithDiff[*beatcrd.Filebeat, client.Object]
}

func newTlsReconciler(c client.Client, recorder record.EventRecorder, log *logrus.Entry) multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, client.Object]

// Read short-circuits the saga when PKI is disabled or has no TLS entries;
// otherwise performs legacy-CA cleanup and delegates to the saga.
func (r *tlsReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (multiphase.MultiPhaseRead[client.Object], reconcile.Result, error)

// filebeatNodeSpecProvider implements pernode.NodeSpecProvider[*beatcrd.Filebeat];
// "nodes" are the keys of spec.pki.tls.
type filebeatNodeSpecProvider struct{}
func (p *filebeatNodeSpecProvider) ExpectedNodeNames(o *beatcrd.Filebeat) ([]string, error)
func (p *filebeatNodeSpecProvider) NodeCertSpec(o *beatcrd.Filebeat, nodeName string) (cn string, dnsNames []string, ips []string, err error)

// filebeatTLSSpec builds the shared base TLSSpec (subject, validity, renewal,
// RSA key size). Per-node CN/DNS/IPs are supplied by NodeCertSpec.
func filebeatTLSSpec(o *beatcrd.Filebeat) certificate.TLSSpec

// filebeatConvergenceCheck gates Rotate→Converge on the Filebeat StatefulSet
// having rolled to the latest generation.
func filebeatConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*beatcrd.Filebeat]

// cleanupLegacyFilebeatCASecret deletes the pre-saga PKI secret <name>-pki-fb once
// the new saga CA secret <name>-tls-fb-ca exists. Best-effort.
func cleanupLegacyFilebeatCASecret(ctx context.Context, c client.Client, o *beatcrd.Filebeat, logger *logrus.Entry) error
```

`newTlsReconciler` body:

```go
saga := rotation.NewTLSStep[*beatcrd.Filebeat](
    c,
    TlsPhase,
    TlsCondition,
    recorder,
    common.FieldManager,
    pernode.NewPerNodeBackend[*beatcrd.Filebeat](&filebeatNodeSpecProvider{}),
    certificate.TLSSpecProviderFunc[*beatcrd.Filebeat](filebeatTLSSpec),
    rotation.WithConvergenceCheck(filebeatConvergenceCheck(c)),
    rotation.WithForceRegenerateAllAnnotation[*beatcrd.Filebeat](AnnotationForceRenewTLS),
    rotation.WithForceRegenerateLeafAnnotation[*beatcrd.Filebeat](AnnotationForceRenewCertificates),
    rotation.WithLabelsDecorator(func(o *beatcrd.Filebeat, obj client.Object) {
        obj.SetLabels(getLabels(o))
    }),
    rotation.WithAnnotationsDecorator(func(o *beatcrd.Filebeat, obj client.Object) {
        obj.SetAnnotations(getAnnotations(o))
        obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
    }),
)
return &tlsReconciler{WorkflowStepReconcilerActionWithDiff: saga}
```

`Read` short-circuit:

```go
if !o.Spec.Pki.IsEnabled() || len(o.Spec.Pki.Tls) == 0 {
    return multiphase.NewMultiPhaseRead[client.Object](), reconcile.Result{}, nil
}
if err := cleanupLegacyFilebeatCASecret(ctx, r.Client(), o, logger); err != nil {
    logger.Warnf("Failed to clean up legacy Filebeat CA secret: %s", err.Error())
}
return r.WorkflowStepReconcilerActionWithDiff.Read(ctx, o, data, logger)
```

`ExpectedNodeNames` / `NodeCertSpec` (deterministic order; mirrors old
`generateCertificate`):

```go
func (p *filebeatNodeSpecProvider) ExpectedNodeNames(o *beatcrd.Filebeat) ([]string, error) {
    names := make([]string, 0, len(o.Spec.Pki.Tls))
    for name := range o.Spec.Pki.Tls {
        names = append(names, name)
    }
    sort.Strings(names)
    return names, nil
}

func (p *filebeatNodeSpecProvider) NodeCertSpec(o *beatcrd.Filebeat, nodeName string) (string, []string, []string, error) {
    tlsSpec, ok := o.Spec.Pki.Tls[nodeName]
    if !ok {
        return "", nil, nil, errors.Errorf("unknown TLS entry %q", nodeName)
    }
    dnsNames := make([]string, 0, (len(o.Spec.Services)*7)+len(o.Spec.Ingresses))
    for _, service := range o.Spec.Services {
        dnsNames = append(dnsNames,
            GetServiceName(o, service.Name),
            fmt.Sprintf("%s.%s", GetServiceName(o, service.Name), o.Namespace),
            fmt.Sprintf("%s.%s.svc", GetServiceName(o, service.Name), o.Namespace),
            GetGlobalServiceName(o),
            fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
            fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
            fmt.Sprintf("*.%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
        )
    }
    for _, ingress := range o.Spec.Ingresses {
        for _, endpoint := range ingress.Spec.Rules {
            dnsNames = append(dnsNames, endpoint.Host)
        }
    }
    dnsNames = append(dnsNames, tlsSpec.AltNames...)

    ips := make([]string, 0, len(tlsSpec.AltIps))
    for _, ipStr := range tlsSpec.AltIps {
        if net.ParseIP(ipStr) == nil {
            return "", nil, nil, errors.Errorf("IP %s is not valid", ipStr)
        }
        ips = append(ips, ipStr)
    }
    return nodeName, dnsNames, ips, nil
}
```

`filebeatTLSSpec`:

```go
func filebeatTLSSpec(o *beatcrd.Filebeat) certificate.TLSSpec {
    validityDays := 365
    if o.Spec.Pki.ValidityDays != nil { validityDays = *o.Spec.Pki.ValidityDays }
    renewalDays := 30
    if o.Spec.Pki.RenewalDays != nil { renewalDays = *o.Spec.Pki.RenewalDays }
    keySize := 2048
    if o.Spec.Pki.KeySize != nil { keySize = *o.Spec.Pki.KeySize }

    return certificate.TLSSpec{
        SecretName:       GetSecretNameForTls(o),
        CommonName:       o.Name,                       // overridden per node by NodeCertSpec
        CACommonName:     fmt.Sprintf("%s-filebeat", o.Name),
        LeafValidityDays: validityDays,
        CAValidityDays:   validityDays,
        RenewalDays:      renewalDays,
        KeyAlgorithm:     certificate.KeyAlgorithmRSA,
        KeySize:          keySize,
        Subject: certificate.CertificateSubject{
            Organizations:       []string{o.Name},
            OrganizationalUnits: []string{"filebeat"},
            Countries:           []string{"internal"},
            Localities:          []string{"internal"},
            Provinces:           []string{"internal"},
        },
    }
}
```

`filebeatConvergenceCheck` (mirrors ES transport, single StatefulSet):

```go
func filebeatConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*beatcrd.Filebeat] {
    return func(ctx context.Context, o *beatcrd.Filebeat, data map[string]any) (bool, error) {
        if common.IsEnvtest() {
            return true, nil
        }
        sts := &appv1.StatefulSet{}
        if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
            if k8serrors.IsNotFound(err) {
                return false, nil
            }
            return false, errors.Wrapf(err, "Error when read Filebeat statefulset")
        }
        if sts.Status.ObservedGeneration < sts.Generation {
            return false, nil
        }
        expectedReplicas := int32(1)
        if sts.Spec.Replicas != nil { expectedReplicas = *sts.Spec.Replicas }
        if sts.Status.CurrentReplicas != expectedReplicas || sts.Status.UpdatedReplicas != expectedReplicas {
            return false, nil
        }
        return true, nil
    }
}
```

`cleanupLegacyFilebeatCASecret`:

```go
func cleanupLegacyFilebeatCASecret(ctx context.Context, c client.Client, o *beatcrd.Filebeat, logger *logrus.Entry) error {
    newCA := &corev1.Secret{}
    if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPki(o)}, newCA); err != nil {
        if k8serrors.IsNotFound(err) { return nil }
        return err
    }
    legacyName := fmt.Sprintf("%s-pki-fb", o.Name)
    legacy := &corev1.Secret{}
    if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: legacyName}, legacy); err != nil {
        if k8serrors.IsNotFound(err) { return nil }
        return err
    }
    if err := c.Delete(ctx, legacy); err != nil { return err }
    logger.Infof("Deleted legacy Filebeat CA secret %s/%s", o.Namespace, legacyName)
    return nil
}
```

### 5.4 Logstash differences (same structure, different names/CN/OU/SANs)

- `AnnotationForceRenewTLS = "logstash.k8s.webcenter.fr/force-renew-tls"`,
  `AnnotationForceRenewCertificates = "logstash.k8s.webcenter.fr/force-renew-certificates"`.
- `nodeSpecProvider` reads `o.Spec.Pki.Tls` of type `map[string]LogstashTlsSpec`.
- `NodeCertSpec` builds DNS names from `o.Spec.Services` **without** the global
  service-name entries (Logstash's old `generateCertificate` only adds
  `GetServiceName(o, service.Name)`, `.%s`, `.%s.svc` per service) + ingress hosts
  + `tlsSpec.AltNames`; IPs from `tlsSpec.AltIps`.
- `logstashTLSSpec`: `CACommonName = fmt.Sprintf("%s-logstash", o.Name)`,
  `OrganizationalUnits = []string{"logstash"}`, `SecretName = GetSecretNameForTls(o)`
  (`<name>-tls-ls`).
- `logstashConvergenceCheck` reads `GetStatefulsetName(o)` and references
  `logstashcrd.Logstash`.
- `cleanupLegacyLogstashCASecret` deletes `<name>-pki-ls` once `<name>-tls-ls-ca`
  exists.

## 6. Reconcile flow integration

`NewFilebeatReconciler` / `NewLogstashReconciler` — replace the TLS step entry only
(position unchanged; TLS runs before CA/credential/configmap/statefulset):

```go
// was:
//   multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret, client.Object](newTlsReconciler(c, recorder)),
// now:
newTlsReconciler(c, recorder, logger),
```

Flow (library code, phase `""`, self-managed PKI enabled):

1. `computeSpec` = `filebeatTLSSpec` (no customizer).
2. Load current leaf `<name>-tls-fb` and CA `<name>-tls-fb-ca` (NotFound → nil).
3. `caNeed = !caExists || forceAll || CANeedsRenewal || CAContentChanged`;
   `leafChg = LeafNeedsChange` (per-node: missing/expiring/CN/org/subject/key/
   usages/node-set drift).
4. `caNeed` → `runCASaga`: new CA + leaf with `ca.crt = newCA||oldCA` bundle,
   publishes `data["rotationRenewed"]=true` and `LayerSignals` under
   `data["tls.Tls"]`; else non-zero `leafChg` → `DesiredLeafWithCA` (leaf-only);
   else steady.
5. `OnSuccess` advances `""→Rotate` when `rotationRenewed`; removes honored force
   annotations after a successful Apply.
6. `OnDiff` (Rotate): `filebeatConvergenceCheck`; not converged →
   `RequeueAfter: workflow.DefaultRequeueAfter`.
7. `Converge`: leaf `ca.crt` = CA `ca.crt` (strip bundle); `OnSuccess` → `""`.

Rollout: the existing StatefulSet builder hashes the whole leaf Secret data
(`<name>-tls-fb`) into `…/secret-<name>`, so the bundle (phase `""`) and the strip
(phase `Converge`) each roll the StatefulSet; no `LayerSignals` wiring is needed.

## 7. RBAC / watches / configuration

- **No RBAC change needed.** Filebeat/Logstash controllers already grant
  `secrets: get;list;watch;create;update;patch;delete` (the saga uses SSA via
  `common.FieldManager`). The new CA Secret `<name>-tls-{fb,ls}-ca` is created under
  the same Secret RBAC.
- **No new watches needed.** Saga-managed Secrets carry no owner reference, so they
  are not driven by `Owns(&corev1.Secret{})`; instead the saga self-drives via CR
  status writes (phase advance) + the convergence-check requeue, exactly as
  Kibana/ES. (Existing `Watches(&corev1.Secret{}, ...)` covers only user-referenced
  Secrets; leave as-is.)
- **Configuration:** tick cadence = saga default requeue (`workflow.DefaultRequeueAfter`)
  during `Rotate`; cert validity/renewal windows from `spec.pki.{validityDays,
  renewalDays, keySize}`; Secret names from the existing helpers; labels/annotations
  from `getLabels`/`getAnnotations` (the Filebeat `…/tls-certificate: "true"` extra
  label is intentionally dropped — it was never consumed by a selector).

## 8. Edge cases & error handling

- **First run / migration of a running deployment:** `caExists=false` (legacy CA is
  at `<name>-pki-{fb,ls}`, a different name) → `caNeed` → full CA saga. Because the
  old leaf Secret exists, `runCASaga` bundles the old `ca.crt` into the new leaf's
  `ca.crt` (newCA||oldCA), so consumers still trusting the old CA keep working
  during the `Rotate` window; `Converge` strips it. Legacy `<name>-pki-*` is deleted
  best-effort once the new CA exists.
- **CA secret missing at `<name>-tls-*-ca`:** treated as "generate" (`CANeedsRenewal`
  returns true for nil); no error.
- **Renewal during rolling upgrade:** leaf Secret data change → StatefulSet hash
  change → full parallel-pod-management rollout; convergence check holds
  `Rotate→Converge` until the StatefulSet reports rolled.
- **Expiry vs rotation windows:** `spec.pki.renewalDays` (default 30) is the single
  window for both CA and leaf (matching today). No CA-specific window (no
  `caRenewalDays` on `PkiSpec`).
- **Leaf-only vs CA rotation:** leaf expiry/CN/SAN/key/node-set drift → leaf-only
  re-sign against existing CA (no CA rotation); CA expiry/content drift → full saga.
- **Node add/remove (`spec.pki.tls` changed):** `LeafNodesChanged` → leaf-only
  regeneration of all nodes → whole-Secret hash change → full StatefulSet rollout
  (functionally identical to today's whole-Secret hash).
- **Corrupt CA/leaf (`ca.crt`/`<node>.crt` unparseable):** `CANeedsRenewal` /
  `LeafNeedsChange` return an error → reconcile error → exponential backoff via
  `common.DefaultControllerRateLimiter`.
- **Empty `spec.pki.tls` with PKI enabled:** short-circuited in `Read` (avoids the
  pernode backend's "at least one expected node" error). Stale Secrets linger on
  subsequent disables — same as today's disabled-PKI path.
- **Invalid `altIps`:** rejected in `NodeCertSpec` with `"IP %s is not valid"`.
- **Force annotations:** honored only at phase `""`, read as `== "true"`,
  force-all wins, removed only after a successful Apply; malformed values ignored.
- **Owner references / GC:** saga Secrets have no owner reference (same as Kibana/ES
  transport). Confirm the multiphase Delete/cleanup path removes
  `<name>-tls-{fb,ls}` and `<name>-tls-{fb,ls}-ca` on CR deletion; if not, the
  legacy `<name>-pki-*` cleanup plus a finalizer-side delete may be needed (flagged
  for verification in §11).

## 9. Risks & open questions

- **Leaf `Organization` field changes** from `<entry name>` to `<CR name>` on first
  reconcile (one-time `LeafOrgChanged`). Trust is CA-anchored, so no consumer
  break; but note it in release notes.
- **Owner references dropped** on `<name>-tls-*` and `<name>-tls-*-ca` (assertion
  change in tests). Verify CR-delete cleanup still removes them (Kibana already
  relies on this; confirm the same for Filebeat/Logstash multiphase Delete path).
- **`ca.pub`/`ca.crl` dropped** (goca-specific; no consumer).
- **Double rollout during CA rotation** (`""` bundle + `Converge` strip) — inherent
  to the saga, same as Kibana; acceptable.
- **`GetSecretNameForPki` semantic change is cross-component:** Filebeat's
  `secret_ca_logstash_reconciler.go` calls `logstashcontrollers.GetSecretNameForPki(ls)`
  and will transparently read the new `<name>-tls-ls-ca`. Until Logstash's saga first
  runs, Filebeat requeues (30s) — eventually consistent; no code change required.
- **Empty-`tls`/disabled-PKI stale Secrets** are not garbage-collected (no owner
  refs); acceptable, matches today.

## 10. Ordered task list

1. `api/beat/v1/filebeat_types.go` + `filebeat_func.go`: add `TlsWorkflowStatus` +
   `GetWorkflowStatus()` + `workflow` import.
2. `api/logstash/v1/logstash_types.go` + `logstash_func.go`: same.
3. `internal/controller/filebeat/helper.go` and `logstash/helper.go`: change
   `GetSecretNameForPki` → `<name>-tls-fb-ca` / `<name>-tls-ls-ca`.
4. Rewrite `internal/controller/filebeat/secret_tls_reconciler.go` per §5.3.
5. Rewrite `internal/controller/logstash/secret_tls_reconciler.go` per §5.4.
6. Delete `internal/controller/{filebeat,logstash}/secret_tls_builder.go` and
   `.../secret_tls_builder_test.go`.
7. `internal/controller/filebeat/filebeat_controller.go` +
   `logstash/logstash_controller.go`: swap TLS step entry (pass `logger`).
8. `pkg/pki/common.go` + `common_test.go`: remove `LoadRootCA` + goca; run
   `go mod tidy`.
9. `make generate` (deepcopy) and `make manifests` (CRD).
10. Update helper tests + controller tests; add new reconciler unit tests (§12).
11. Run validation (§11).

## 11. Validation / verification commands

```
make generate manifests        # deepcopy + CRD (status.tlsWorkflowStatus)
go mod tidy                    # drops github.com/disaster37/goca
go build ./...
go vet ./...
make test                      # envtest (KUBEBUILDER_ASSETS + ES_OPERATOR_ENVTEST=true)
#   == go test -p 1 -v -timeout 1200s ./api/... ./internal/controller/... ./pkg/...
```
Targeted (fast) checks:
```
go test ./api/beat/... ./api/logstash/... ./internal/controller/filebeat/... ./internal/controller/logstash/... ./pkg/pki/...
```
No `golangci-lint` config is present in the repo; skip unless CI introduces one.

## 12. Test plan

Framework: `testify/suite` + envtest (existing convention; `suite_test.go`). Unit
tests are plain `testing` + `testify/assert`.

### 12.1 Unit tests (no envtest)

- **`internal/controller/filebeat/secret_tls_reconciler_test.go`** (new):
  - `filebeatTLSSpec`: defaults (SecretName `<name>-tls-fb`, CA CN `<name>-filebeat`,
    OU `filebeat`, validity 365 / renewal 30 / RSA 2048); overrides from
    `ValidityDays`/`RenewalDays`/`KeySize`.
  - `filebeatNodeSpecProvider.ExpectedNodeNames`: sorted keys of `spec.pki.tls`.
  - `filebeatNodeSpecProvider.NodeCertSpec`: CN == node name; DNS list includes
    service + global-service + ingress-host + altNames; `altIps` parsed; invalid IP
    → error.
- **`internal/controller/logstash/secret_tls_reconciler_test.go`** (new): same,
  with Logstash-specific DNS list (no global-service entries), OU `logstash`, CA CN
  `<name>-logstash`.
- **`internal/controller/{filebeat,logstash}/helper_test.go`**: update
  `TestGetSecretNameForPki` expected values.
- **`pkg/pki/common_test.go`**: drop the `LoadRootCA` test (goca removal).

### 12.2 envtest integration (extend existing controller tests)

- **Bootstrap (existing `TestFilebeatController` / `TestLogstashController`):**
  replace `assert.NotEmpty(s.OwnerReferences)` for the PKI and TLS Secrets with
  `assert.NotEmpty(s.Data["ca.crt"])` (CA Secret also `assert.NotEmpty(s.Data["ca.key"])`;
  leaf Secret asserts `ca.crt` + one `<entry>.crt`/`<entry>.key` per `spec.pki.tls`).
  Assert `status.tlsWorkflowStatus.currentPhase` converges to `""`.
- **CA rotation (new):** set `Annotations[AnnotationForceRenewTLS]="true"`; assert CA
  Secret `ca.crt` changed and annotation cleared.
- **Leaf renew (new):** set `Annotations[AnnotationForceRenewCertificates]="true"`;
  assert leaf `<entry>.crt` changed while CA `ca.crt` unchanged; annotation cleared.
- **Node add/remove (new):** add/remove a `spec.pki.tls` entry; assert the leaf
  Secret gains/loses `<entry>.crt`/`<entry>.key` and the StatefulSet rolls.
- **Disabled PKI (new):** `spec.pki.enabled=false` → no `<name>-tls-*` /
  `<name>-tls-*-ca` Secrets created and no phase writes.
- **Filebeat↔Logstash CA mirror (existing):** confirm Filebeat's
  `GetSecretNameForCALogstash` still populates from the new
  `<name>-tls-ls-ca` (`ca.crt`) after Logstash converges.

### 12.3 Manual smoke (optional)

`make run`; create a Filebeat/Logstash CR with `spec.pki.tls`; watch
`<name>-tls-*` / `<name>-tls-*-ca`; apply each force annotation and confirm
rotation/renewal + annotation cleanup + StatefulSet rollout.
