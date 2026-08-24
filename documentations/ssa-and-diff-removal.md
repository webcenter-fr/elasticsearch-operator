# SSA and diff removal

This document explains why the client-side 3-way strategic-merge diff was removed
from the multiphase reconcilers during the `operator-sdk-extra` v2 → v3 migration
and how field ownership is managed instead.

## 1. What changed

The multiphase and sentinel reconcilers moved from a client-side 3-way
strategic-merge diff (powered by `github.com/disaster37/k8s-objectmatcher`) to
Kubernetes **Server-Side Apply (SSA)**.

- v2: expected objects were compared against the live object with
  `patch.DefaultPatchMaker.Calculate(...)`, and the resulting merge patch was
  applied with `Create()` / `Update()`. The
  `kubectl.kubernetes.io/last-applied-configuration` annotation was written by
  `patch.DefaultAnnotator.SetLastAppliedAnnotation(...)` to drive the 3-way merge.
- v3: every step reconciler declares its desired objects and applies them with
  `client.Apply` using a single stable `fieldManager`
  (`common.FieldManager = "elasticsearch-operator"`) and
  `client.ForceOwnership`. The `Diff()` method now classifies objects into
  create/update/delete lists via an SSA dry-run (diff variant) or an
  always-apply classification (simple variant); `Create()`/`Update()` are
  replaced by a single `Apply()`.

## 2. Why the diff options and last-applied annotation were removed

- `DiffPathOption`, `DiffFunc`, `IgnoreUnset`, `ignoreDiff ...patch.CalculateOption`
  and `patch.DefaultAnnotator.SetLastAppliedAnnotation` were all mechanisms built
  around the client-side 3-way merge. SSA keeps per-field ownership in
  `metadata.managedFields`: fields the operator does not set are simply not
  applied, so "ignore field" options are unnecessary.
- The `kubectl.kubernetes.io/last-applied-configuration` annotation is no longer
  used by the multiphase/sentinel patterns. SSA does not need it to compute
  ownership, and the legacy annotation is cleaned up from managed children by
  `multiphase.CleanupReadLastAppliedAnnotations`.

## 3. Fields the operator must not own

The operator must omit fields it does not own from its expected objects. SSA
leaves other managers' fields intact for fields it does not apply. Residual
conflicts are resolved with `client.ForceOwnership`; genuine `Conflict` errors
are surfaced and retried by controller-runtime backoff.

Requirements for SSA expected objects:

- `TypeMeta` (`apiVersion`/`kind`) must be set (required by the SSA dry-run
  classification used by the diff variant).
- deterministic `Name` (no `generateName`).
- no status fields in expected objects.

## 4. Why `generic-objectmatcher` (remote pattern) is retained

The remote reconciler pattern (Elasticsearch/Kibana API objects) reconciles
**external API objects**, not Kubernetes objects. SSA field ownership does not
apply to external REST APIs, so v3's remote pattern still uses
`generic-objectmatcher/patch` 3-way merge together with the
`status.lastAppliedConfiguration` bookkeeping.

Files that still use `generic-objectmatcher/patch`:

- `internal/controller/elasticsearchapi/*_api_client.go` and `*_reconciler.go`
- `internal/controller/kibanaapi/*_api_client.go` and `*_reconciler.go`

## 5. Upgrade path

Objects previously managed via update/merge + `last-applied-configuration` are
re-owned on the first SSA apply (`client.ForceOwnership`). The legacy
`kubectl.kubernetes.io/last-applied-configuration` annotation is removed from
managed children by `multiphase.CleanupReadLastAppliedAnnotations`, which runs
before every diff. Conflicting external field managers (e.g. `kubectl`, HPA) are
overridden for the fields the operator owns.
