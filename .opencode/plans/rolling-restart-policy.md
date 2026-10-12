# Rolling Restart Policy (`spec.updateStrategy.rollingRestartPolicy`)

## Goal

Let users choose how the operator prepares an Elasticsearch cluster before a
rolling restart, via a new CRD field, without breaking current behavior:

- **`graceful` (default)** — Node Shutdown API strategy when ES >= 7.15.2
  (current "safe" behavior; graceful shard migration off each node). Falls back
  to the allocation-filter strategy on older versions.
- **`fast` (new)** — force the allocation-filter strategy on **every** version:
  transient `cluster.routing.allocation.enable=primaries` +
  `cluster.routing.rebalance.enable=none` + flush. No shard migration: faster,
  uses less disk, but relies on already-started replicas (less robust).

Field absent -> current behavior (backward compatible).

## Naming decision (final)

**Field path:** `spec.updateStrategy.rollingRestartPolicy`
**Enum values:** `graceful` (default) / `fast`

Rationale:
- `updateStrategy` is a nestable, extensible home for upgrade knobs (future:
  StatefulSet updateStrategy type, maxUnavailable, etc.) and mirrors the
  Kubernetes `updateStrategy` concept without clashing with it.
- `graceful` matches the codebase's existing wording ("graceful shard
  migration", "Requesting graceful shutdown") and precisely names the safe
  mode's behavior.
- `fast` names the concrete user benefit (faster restarts, less disk), not the
  mechanism, and carries no legacy/deprecated connotation (unlike
  "allocation-filter" or "legacy"). It leaves room for a future middle mode.

## File-by-file changes

### 1. `api/elasticsearch/v1/elasticsearch_types.go`

Add the policy type + constants (next to the other spec types, after the
`const` block at the top of the file):

```go
// RollingRestartPolicy controls how the operator prepares the cluster before
// restarting pods during a rolling restart.
// +kubebuilder:validation:Enum=graceful;fast
type RollingRestartPolicy string

const (
	// RollingRestartPolicyGraceful is the default: use the Node Shutdown API
	// on ES >= 7.15.2 (graceful shard migration) and fall back to the
	// allocation-filter approach on older versions.
	RollingRestartPolicyGraceful RollingRestartPolicy = "graceful"
	// RollingRestartPolicyFast forces the allocation-filter approach on every
	// version: transient cluster.routing.allocation.enable=primaries +
	// cluster.routing.rebalance.enable=none + flush. Faster and uses less
	// disk, but does not migrate shards gracefully.
	RollingRestartPolicyFast RollingRestartPolicy = "fast"
)
```

Add a nested struct and a field on `ElasticsearchSpec` (append
`UpdateStrategy` after the `Monitoring` field):

```go
// ElasticsearchUpdateStrategySpec configures how the operator performs
// cluster upgrades.
type ElasticsearchUpdateStrategySpec struct {
	// RollingRestartPolicy controls how pods are restarted during a rolling
	// upgrade.
	// - graceful (default): Node Shutdown API on ES >= 7.15.2, allocation
	//   filter on older versions.
	// - fast: always use the allocation filter (transient
	//   cluster.routing.allocation.enable=primaries +
	//   cluster.routing.rebalance.enable=none + flush).
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=graceful
	RollingRestartPolicy RollingRestartPolicy `json:"rollingRestartPolicy,omitempty"`
}
```

```go
	// UpdateStrategy configures how the operator performs cluster upgrades.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	UpdateStrategy ElasticsearchUpdateStrategySpec `json:"updateStrategy,omitempty"`
```

### 2. `api/elasticsearch/v1/zz_generated.deepcopy.go`

Regenerated (do not edit by hand). `make generate` will add a
`DeepCopyInto`/`DeepCopy` for `ElasticsearchUpdateStrategySpec` (a plain
`*out = *in` copy) and copy `UpdateStrategy` in `ElasticsearchSpec.DeepCopyInto`
(already covered by the leading `*out = *in`).

### 3. `internal/controller/elasticsearch/rolling_restart.go`

Add a `policy` field (+ optional `strategyReason`) to the struct:

```go
type RollingRestartOrchestrator struct {
	esHandler      eshandler.ElasticsearchHandler
	k8sClient      client.Client
	logger         *logrus.Entry
	strategy       RollingRestartStrategy
	policy         elasticsearchcrd.RollingRestartPolicy
	strategyReason string // human-readable fallback reason; empty when none
	esVersion      string
	nodeNameToID   map[string]string
}
```

Add a pure, unit-testable resolver:

```go
// resolveRollingRestartStrategy maps the user-requested policy and detected
// Elasticsearch version to the effective strategy. It returns a human-readable
// reason when the request cannot be honored (fallback), or "" when none.
func resolveRollingRestartStrategy(policy elasticsearchcrd.RollingRestartPolicy, version string) (RollingRestartStrategy, string)
```

Resolver semantics (matrix):

| policy | ES >= 7.15.2 | ES < 7.15.2 | version "" / unparseable |
|---|---|---|---|
| `graceful` (or `""`, absent) | `node-shutdown`, reason="" | `allocation-filter`, reason="...does not support Node Shutdown API..." | `allocation-filter`, reason="...does not support..." |
| `fast` | `allocation-filter`, reason="" (forced) | `allocation-filter`, reason="" | `allocation-filter`, reason="" |
| any other value | treat as `graceful` + warning reason="unknown policy %q; treating as graceful" | (same graceful fallback) | (same graceful fallback) |

Implementation notes:
- Normalize: `case "", elasticsearchcrd.RollingRestartPolicyGraceful: ...`,
  `case elasticsearchcrd.RollingRestartPolicyFast: ...`, `default: ...`.
- Decision for `graceful` delegates to existing `supportsNodeShutdown(version)`.
- `fast` never calls `supportsNodeShutdown`.

Change the constructor signature and logic:

```go
func NewRollingRestartOrchestrator(
	esHandler eshandler.ElasticsearchHandler,
	k8sClient client.Client,
	logger *logrus.Entry,
	policy elasticsearchcrd.RollingRestartPolicy,
) (*RollingRestartOrchestrator, error)
```

Constructor logic:
- `fast` (or resolution ends in `allocation-filter` and version is not
  required): **skip** `GetVersion` and `GetNodeNameToIDMap` entirely; set
  `esVersion=""`, `nodeNameToID=map[string]string{}`.
- `graceful`: call `GetVersion` (error still returns the wrapped error, same as
  today). Resolve strategy. Fetch `GetNodeNameToIDMap` **only** when strategy
  is `node-shutdown` (keep the existing warn-and-continue behavior); skip it
  for `allocation-filter`.
- Log at INFO (satisfies the "clear INFO logs" requirement):

```text
Rolling restart policy from spec: %s
Rolling restart strategy resolved: %s (ES version: %q)%s
```
  where the trailing `%s` is `"; " + strategyReason` when non-empty, else "".

Add accessors:

```go
func (r *RollingRestartOrchestrator) Policy() elasticsearchcrd.RollingRestartPolicy
func (r *RollingRestartOrchestrator) Strategy() RollingRestartStrategy // existing
```

Modify `CompleteRollingRestart` — in the `StrategyNodeShutdown` branch, before
the existing shutdown cleanup, add a defensive transient-settings clear:

```go
// Node Shutdown path: defensively clear any transient allocation settings
// left over from a previous/interrupted "fast" run (or a mid-upgrade policy
// switch graceful<->fast).
r.logger.Info("Removing any leftover transient allocation settings")
if err := r.esHandler.RemoveTransientAllocationSettings(); err != nil {
	r.logger.Warnf("Failed to remove transient allocation settings (non-blocking): %s", err)
}
```

Leave `PrepareForRollingRestart` unchanged except optionally log the resolved
policy/strategy (the constructor already logs it). Keep the existing strategy
branch logs.

### 4. `internal/controller/elasticsearch/statefulset_reconciler.go`

Thread the policy from the CR at both call sites (both have `o` in scope):

- `Diff()` completion path (~line 403):
  `NewRollingRestartOrchestrator(esHandler, r.Client(), logger, o.Spec.UpdateStrategy.RollingRestartPolicy)`
- `checkUpgradePredicates()` (~line 475):
  `NewRollingRestartOrchestrator(esHandler, r.Client(), logger, o.Spec.UpdateStrategy.RollingRestartPolicy)`

No other call sites exist (verified).

### 5. `config/crd/bases/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml`

Regenerated via `make manifests`. Expect a new `updateStrategy` object under
`spec.properties` with `rollingRestartPolicy` having `default: graceful` and
`enum: [graceful, fast]`.

The `manifests` target runs `hack/strip-crd-cel-validations` afterward; enum and
default markers are not CEL validations, so this does not affect the new field.

### 6. `bundle/manifests/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml` (+ CSV descriptions)

Regenerated via `make bundle`. This also updates
`config/manifests/bases/elasticsearch-operator.clusterserviceversion.yaml`
(`+operator-sdk:csv:customresourcedefinitions:type=spec` marker drives the
operator-sdk spec descriptors). Note: `make bundle` requires `operator-sdk` +
`kustomize` on PATH and can touch CSV `spec.version`/`spec.replaces`.

## CRD regeneration targets (verified in Makefile)

- `make generate` — deepcopy (`controller-gen object:...`).
- `make manifests` — CRD manifests (`controller-gen ... crd:...`) + CEL strip.
- `make bundle` — OLM bundle manifests + CSV (heavier; requires operator-sdk).
- `make generate-json-schema` — only if a local `json-schema/` dir is wanted
  (none is committed today, so skip).

## Effective strategy resolution (summary)

On every reconcile the orchestrator is rebuilt from the **current** CR spec +
detected version (there is no persisted "mode" state). Resolution:
`graceful` -> node-shutdown iff `supportsNodeShutdown(version)` else
allocation-filter; `fast` -> allocation-filter always. Absent/empty/unknown
policy -> `graceful`.

## Edge cases & error handling

- **Field absent / `""`**: resolves to `graceful` => current behavior. No
  migration.
- **Unknown policy value**: rejected at admission by the CRD enum; defensively
  treated as `graceful` with a WARN/INFO log in `resolveRollingRestartStrategy`.
- **ES version unknown / unparseable** (`"latest"`, `"7"`, `""`, `"abc"`):
  `supportsNodeShutdown` returns false -> allocation-filter fallback with an
  INFO reason. `fast` doesn't care about version.
- **`GetVersion` API error**: only reachable on `graceful`; constructor returns
  the wrapped error (unchanged). On `fast` no version call is made, so no such
  error.
- **nil esHandler**: unchanged — `Diff()` completion already errors; the start
  path already skips orchestration and logs a warning.
- **Policy switched mid-upgrade**: `Prepare` ran with the old policy, `Complete`
  runs with the new one. The defensive `RemoveTransientAllocationSettings()` in
  the node-shutdown `Complete` branch guarantees no stranded
  `primaries`/`none`. `fast`'s own `Complete` re-enables as before.
- **Leftover transient settings**: `fast` `Complete` re-enables via
  `EnableShardAllocationTransient` + `EnableRoutingRebalanceTransient`; graceful
  `Complete` now defensively `RemoveTransientAllocationSettings()`.
- **Field removed from a CR**: same as absent => graceful default.

## Logging (INFO)

- `Rolling restart policy from spec: <graceful|fast|(empty)>`
- `Rolling restart strategy resolved: <node-shutdown|allocation-filter> (ES version: <v>)<; reason>`
- Fallback reason examples:
  `Elasticsearch version "7.14.0" does not support the Node Shutdown API (requires >= 7.15.2); falling back to allocation-filter`
  and `unknown rolling restart policy "bogus"; treating as graceful`.
- `Removing any leftover transient allocation settings` (graceful complete).
- Existing `Prepare`/`Complete` INFO logs remain.

## Tests

### Unit tests — `internal/controller/elasticsearch/rolling_restart_test.go`
(No envtest needed; uses gomock `mocks.NewMockElasticsearchHandler`.)

- `TestResolveRollingRestartStrategy` — table-driven `(policy, version) ->
  (strategy, reasonContains)` covering the full matrix (graceful/fast/""/bogus
  x 8.11.0/7.14.0/"", "invalid").
- `TestNewRollingRestartOrchestratorPolicy`:
  - `fast` -> strategy `allocation-filter`; expect **no** `GetVersion` /
    `GetNodeNameToIDMap` calls (gomock).
  - `graceful` + `"8.11.0"` -> `node-shutdown`; expects `GetVersion` -> `"8.11.0"`
    and `GetNodeNameToIDMap` -> map.
  - `graceful` + `"7.14.0"` -> `allocation-filter`; expects `GetVersion` and
    **no** `GetNodeNameToIDMap`.
  - `graceful` + `GetVersion` error -> constructor returns error containing
    "failed to get Elasticsearch version".
- `TestCompleteRollingRestart` — **update the existing node-shutdown subtest**
  to also expect `mockES.EXPECT().RemoveTransientAllocationSettings().Return(nil)`
  (before `GetAllNodeShutdowns`), or gomock will fail on the unexpected call.

### Existing tests affected
- `TestRollingRestartOrchestratorAccessors` — constructs the struct literal
  directly; still compiles (new fields default), optionally assert `Policy()`.
- `TestPrepareForRollingRestart`, `TestRequestNodeShutdown`,
  `TestWaitForNodeShutdown`, `TestClearNodeShutdown`, `TestCheckPredicates`,
  `TestSupportsNodeShutdown` — unchanged (they build struct literals; no
  constructor call).

### Golden files
No changes: the new field does not affect StatefulSet pod templates or any
built object (`internal/controller/elasticsearch/testdata/statefullset-*.yml`
unchanged).

### Envtest
`internal/controller/elasticsearch/suite_test.go` runs the controller suite and
needs `KUBEBUILDER_ASSETS` (via `make test`, which downloads envtest). The unit
tests above run without envtest. Regenerating the CRD is sufficient for envtest
to admit the new field (it defaults to `graceful`); no new envtest test is
required, but verify `make test` still passes.

## Docs

- `documentations/rolling-upgrade.md`:
  - Rename/rewrite "Version-aware strategy" -> "Rolling restart policy".
  - Describe `graceful` vs `fast` with a precise trade-off table
    (shard migration: yes/no; disk usage: higher/lower; speed: slower/faster;
    robustness: graceful migration vs relies-on-started-replicas).
  - Document the version fallback for `graceful` on ES < 7.15.2.
  - Add a YAML example with `spec.updateStrategy.rollingRestartPolicy: fast`.
  - Update the "Logs" section with the new INFO lines.
- `documentations/elasticsearch/main-settings.md`: add a bullet for
  `updateStrategy.rollingRestartPolicy` (default `graceful`).

## Implementation order

1. Edit `elasticsearch_types.go` (type + constants + struct + field).
2. `make generate` (deepcopy).
3. Edit `rolling_restart.go` (resolver + constructor + struct fields +
   `Complete` defensive remove).
4. Edit `statefulset_reconciler.go` (thread policy at 2 call sites).
5. Update/add tests in `rolling_restart_test.go` (incl. the modified existing
   node-shutdown `Complete` mock).
6. `make manifests` (CRD). Verify the new field/enum/default appear.
7. `make bundle` (if bundle artifacts are committed in this change).
8. Update docs (2 files).
9. Run verification.

## Verification commands

```bash
cd /projects/elasticsearch-operator-worktrees/eck-lifecycle-backport
make generate manifests        # deepcopy + CRD; confirm diff is scoped
gofmt -l ./...                 # no output expected (or run `make fmt`)
go build ./...
go vet ./...
go test ./internal/controller/elasticsearch/ \
  -run 'TestResolveRollingRestartStrategy|TestNewRollingRestartOrchestratorPolicy|TestCompleteRollingRestart|TestSupportsNodeShutdown|TestPrepareForRollingRestart|TestRollingRestartOrchestratorAccessors' \
  -count=1
# full suite (downloads/uses envtest):
make test
# confirm CRD shape:
grep -nA10 'updateStrategy' config/crd/bases/elasticsearch.k8s.webcenter.fr_elasticsearches.yaml
```

## Backward compatibility

- Absent/empty field == `graceful` == current auto-version behavior. No data or
  schema change required for existing CRs.
- The default marker (`graceful`) is transparently applied by the API server to
  newly written objects; existing stored objects without the field behave
  identically.
- The defensive `RemoveTransientAllocationSettings()` in the graceful-complete
  path is idempotent (PUT `null` on already-unset transient keys is a no-op) and
  only adds a single extra API call.

## Licensing

Concept (two rolling-restart strategies) is ECK-inspired but no ECK code is
copied (ECK is Elastic License 2.0; this project is Apache 2.0). All new code is
original and consistent with the existing header comment in
`rolling_restart.go`.

## Open questions / out of scope

- Per-pod Node Shutdown request loop (register `restart` before pod deletion)
  remains not-yet-enabled (StatefulSets still use `RollingUpdate`); `fast`
  only affects the cluster-level prepare/complete. Out of scope here.
- No new `maxUnavailable`/StatefulSet updateStrategy exposure — future
  extensibility only.