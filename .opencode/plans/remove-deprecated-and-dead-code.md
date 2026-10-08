# Remove Genuinely Deprecated and Dead Code — Implementation Plan

## 1. Summary & Objectives

This plan removes **genuinely deprecated** code (symbols the Go toolchain actually flags with `// Deprecated:` / staticcheck SA1019) and **genuinely dead** code (truly unused symbols/files, staticcheck U1000 / `unused`) from the `github.com/webcenter-fr/elasticsearch-operator` Go codebase (Go 1.27, `controller-runtime v0.24.1`, `k8s.io/* v0.36.4`, `logrus v1.10.1`, `emperror.dev/errors v0.8.1`, `go-funk v0.9.3`, `disaster37/*` helpers).

**In scope**
1. Restore the five typo-renamed `secret_ca_*_builder_test..go` dead test files to their correct `*_test.go` names (they are uncompiled and therefore dead), fixing any stale test assertions that surface after restoration.
2. Fix any real `// Deprecated:` / SA1019 usage discovered by `staticcheck` — **replaced with the documented replacement provided by the library/standard library** (never hand-rolled). Candidate surfaces: `disaster37/*` helper libraries, `k8s.io/*`, `controller-runtime`, `logrus`, and the Go standard library.
3. Remove any truly-unused symbol/file discovered by `staticcheck -checks=U1000` (or `golangci-lint` `unused`), subject to the explicit exclusions in §3.D.

**Out of scope (explicitly KEPT — do not touch)**
- `emperror.dev/errors` — **not deprecated, intentionally used, stays.** No migration, no `go.mod` removal.
- `github.com/thoas/go-funk` — **not deprecated, intentionally used, stays.** No migration, no `go.mod` removal.
- `scripts/clean-crd.go` and the `cleanCrdVersion` constant in `ci/dagger/main.go` — **preserved.** They are useful for the occasional manual CRD-update cleanup and are deliberately kept.
- Any change that alters runtime behavior, resource naming, reconciliation logic, or CRD schemas.
- Hand-editing generated files (`zz_generated.deepcopy.go`, `ci/dagger/dagger.gen.go`, CRD/config/bundle YAML).
- Upgrading dependencies to newer major versions.

**Guiding principle:** always use an existing library's own documented replacement API rather than recreating/duplicating its behavior in local code. When staticcheck flags a deprecated symbol, apply the library's documented replacement; do **not** write a local helper that mimics library functionality.

---

## 2. Detection Methodology

Detection is linter-driven. `go vet`/`go build` do **not** flag most deprecations (they are doc-comment markers), so the primary tools are `staticcheck` (SA1019 deprecated usage, U1000 unused) and `golangci-lint` (`unused`/`staticcheck`).

### 2.1 Install verification tools (if absent)

```bash
# staticcheck — primary detector for SA1019 (deprecated) and U1000 (unused)
go install honnef.co/go/tools/cmd/staticcheck@latest

# golangci-lint — bundles unused/staticcheck/ineffassign/govet
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Check availability first: `command -v staticcheck golangci-lint`. If neither is installed and `go install` is unavailable, fall back to §2.4 manual grep.

### 2.2 Run the detectors

```bash
# Full scan (deprecated + unused)
staticcheck ./...

# Deprecated usage only — this is the authoritative "deprecated code" list
staticcheck -checks=SA1019 ./...

# Unused symbols only — this is the authoritative "dead code" list
staticcheck -checks=U1000 ./...

# golangci-lint equivalent
golangci-lint run --enable=unused,staticcheck,ineffassign,govet ./...
```

Note: `golangci-lint`'s `deadcode` linter was retired (superseded by `unused`). Do not rely on `--enable=deadcode`.

Interpretation:
- **SA1019** → a genuinely deprecated symbol is used. Read the `// Deprecated:` text in the flagged symbol's doc to find the documented replacement and apply it.
- **U1000** → an unused symbol. Apply the exclusions in §3.D before deleting.

### 2.3 Compiler / vet baseline (always run, no install needed)

```bash
go vet ./...
go build ./...
gofmt -l .
```

These are the safety net for any edits (unused imports after removal, signature breaks).

### 2.4 Manual fallback (no staticcheck/golangci-lint available)

Run these greps in project source. They encode the manual findings already established in §3.

```bash
# Deprecated stdlib patterns — none currently present; re-verify
rg -n 'ioutil\.|math/rand|golang.org/x/net/context|strings\.Title|bytes\.Title|reflect\.PtrTo|os\.SEEK_'

# Deprecated controller-runtime global logging — VERIFIED NOT DEPRECATED in v0.24.1 (§3.E)
rg -n 'ctrl\.SetLogger|ctrl\.Log|logf\.SetLogger|log\.SetLogger'

# Deprecated handler — VERIFIED NOT DEPRECATED in v0.24.1 (aliases to typed forms, §3.E)
rg -n 'handler\.MapFunc|EnqueueRequestsFromMapFunc'

# Deprecated k8s.io/utils/pointer — already migrated to ptr; re-verify
rg -n 'k8s.io/utils/pointer|pointer\.(Bool|String|Int|Int32|Int64)'

# Deprecated beta APIs — none present; re-verify
rg -n 'v1beta1|extensions/v1|PodSecurityPolicy|apiextensions/v1beta1'

# Dead/typo'd files
find . -name '*..go'
```

For "is symbol X unused" checks without staticcheck: `rg -n '\bX\b' --glob '*.go' --glob '!zz_generated*'` — if the only hit is the declaration, it is dead, subject to the exclusions in §3.D (reflection, string literals, struct tags, `+kubebuilder:` markers, `init()`, blank imports).

---

## 3. Complete Findings Inventory

### A. Confirmed dead-code findings (files)

#### FINDING B1 — Five dead test files with typo'd `.go` extension (`*_test..go`)

These files are **not compiled** by the Go toolchain (a filename ending in `..go` does not match the `.go`/`_test.go` patterns), so they are dead. Each contains a real test for a still-existing builder function; they are almost certainly an accidental `sed`/rename artifact.

- `internal/controller/metricbeat/secret_ca_elasticsearch_builder_test..go` — `TestBuildCAElasticsearchSecret` → targets `buildCAElasticsearchSecrets` (exists in `secret_ca_elasticsearch_builder.go:10`)
- `internal/controller/kibana/secret_ca_elasticsearch_builder_test..go` — `TestBuildCAElasticsearchSecret` → `buildCAElasticsearchSecrets` (`secret_ca_elasticsearch_builder.go:10`)
- `internal/controller/logstash/secret_ca_elasticsearch_builder_test..go` — `TestBuildCAElasticsearchSecret` → `buildCAElasticsearchSecrets` (`secret_ca_elasticsearch_builder.go:10`)
- `internal/controller/filebeat/secret_ca_elasticsearch_builder_test..go` — `TestBuildCAElasticsearchSecret` → `buildCAElasticsearchSecrets` (`secret_ca_elasticsearch_builder.go:10`)
- `internal/controller/filebeat/secret_ca_logstash_builder_test..go` — `TestBuildCALogstashSecret` → `buildCALogstashSecrets` (`secret_ca_logstash_builder.go:10`)
- **Category:** dead-code (file-level; uncompiled).
- **Risk:** **needs care.** Two remediation options:
  1. **Restore** by renaming `secret_ca_*_builder_test..go` → `secret_ca_*_builder_test.go` (recommended — restores coverage of real functions). These tests have not run recently; they may fail against current builder behavior and need stale-assertion fixes.
  2. **Delete** if the tests are considered obsolete.
  Recommend (1), then run the specific package tests and fix failures.
- **Recommendation:** restore via `git mv` to the correct name, run `go test ./internal/controller/{metricbeat,kibana,logstash,filebeat}/...`, and fix failing assertions only if the test expectation is stale (do **not** change builder behavior to satisfy tests).

### B. Findings to be discovered by staticcheck/golangci-lint during implementation

The following could not be exhaustively enumerated by manual grep (no shell access to run staticcheck in the research phase). The implementer MUST run `staticcheck ./...` and triage every hit:

1. **SA1019 genuinely-deprecated usages** — the most likely remaining source is inside the `disaster37/*` helper libraries (`operator-sdk-extra/v3`, `k8sbuilder`, `es-handler/v9`, `kb-handler/v8`, `elasticsearch/v9`, `kibana-rest/v8`, `generic-objectmatcher`, `k8s-objectmatcher`), plus `k8s.io/*`, `controller-runtime`, `logrus`, and the stdlib. For each hit, apply the **documented replacement** from the symbol's `// Deprecated:` text. Do not hand-roll equivalents.
2. **U1000 unused exported/unexported symbols** — any unused function/method/type/struct/field/var/const across `api/*`, `internal/controller/*`, `pkg/*`. Manual spot-checks of `pkg/helper`, `pkg/pki`, `internal/controller/common`, and the `elasticsearch`/`metricbeat` helper files found every symbol referenced; the remainder (if any) will be surfaced by U1000. Apply the exclusions in §3.D.

### C. Explicitly PRESERVED (out of scope — do NOT flag or delete)

These were investigated during research and are **deliberately kept**; do not report them as findings:

- **`emperror.dev/errors`** (imported in ~100 files): not `// Deprecated:` marked; intentionally used. **KEEP.**
- **`github.com/thoas/go-funk`** (imported in 9 files): not `// Deprecated:` marked; intentionally used. **KEEP.**
- **`cleanCrdVersion` constant** in `ci/dagger/main.go:33` and **`scripts/clean-crd.go`** (whole file): useful for occasional manual CRD-update cleanup. **KEEP.** (Do not flag `cleanCrdVersion` as U1000-dead even if `staticcheck` reports it; add a `//nolint:staticcheck // kept for manual CRD cleanup` if needed to keep the build clean, per §5.6.)

### D. Dead-code exclusions (never auto-delete)

A symbol/file is **not** dead (and must not be removed) if it is:
- an **exported symbol in `api/**`** (quasi-public CRD type surface — treat as "needs human decision / keep");
- referenced via **reflection, string literals, struct tags, or `+kubebuilder:` markers** (RBAC/webhook/CRD markers, controller names passed to `mgr.GetEventRecorderFor("<name>")`, field-index paths like `"spec.licenseSecretRef.name"`);
- invoked via **`init()`** or **blank imports** (`_ "k8s.io/client-go/plugin/pkg/client/auth"`, `SchemeBuilder.AddToScheme`);
- inside a **generated file** (`zz_generated.*`, `dagger.gen.go`) — regenerate, don't hand-edit;
- used only from `*_test.go` files (test helpers are "used" within their package's test build).

### E. Verified NOT deprecated (so the coder doesn't chase false positives)

Confirmed against controller-runtime v0.24.1 source (fetched during research): the following are **not** `// Deprecated:` marked and must be left alone:

- `ctrl.SetLogger`, `ctrl.Log`, `logf.SetLogger`, `log.SetLogger` (`pkg/log` has no `// Deprecated:` markers).
- `handler.MapFunc`, `handler.EnqueueRequestsFromMapFunc` (they are aliases to the typed forms).
- `zap.Options`, `opts.BindFlags`, `zap.UseFlagOptions`.
- `admission.Validator[T]`, `admission.Defaulter[T]`, `admission.Warnings` (generic forms are current).

Also verified clean: no `ioutil`, no `math/rand` seed/read, no `golang.org/x/net/context`, no `k8s.io/utils/pointer` (already on `k8s.io/utils/ptr`), no `v1beta1`/`extensions/v1`/`PodSecurityPolicy`/`apiextensions/v1beta1` APIs.

---

## 4. Data Structures & Function Signatures Affected

No public struct/interface/function signature of this repository is expected to change.

- The dead-file restoration (B1) changes **file names only**; no function signatures change.
- Any SA1019 fix replaces a deprecated call with its documented replacement at the call site; this should not alter this repository's own public signatures.
- Any U1000 removal deletes a private symbol (or, for exported `api/**` symbols, is deferred to human decision per §3.D).

If a discovered SA1019 fix *does* require touching a signature (unexpected), record the before/after signature in the PR description before implementing.

---

## 5. Edge Cases & Error Handling

1. **Generated files must not be hand-edited.** `api/**/zz_generated.deepcopy.go`, `ci/dagger/dagger.gen.go`, and CRD/config/bundle YAML are regenerated. If a dead/deprecated symbol is detected *inside* generated code, run `make generate` (deepcopy) or re-run the dagger generator instead of editing. Flag these separately from hand-written findings.
2. **Reflection / string-based wiring.** Symbols referenced only via `+kubebuilder:` markers, controller names, field-index paths, or string literals are **not** dead. Do not delete anything whose name appears in a string literal or struct tag.
3. **`init()` side effects.** `internal/controller/common/metric.go` registers metrics in `init()`; `cmd/main.go` registers schemes in `init()`. Do not remove symbols that look "unused" but are invoked via `init()`, blank imports, or `SchemeBuilder.AddToScheme`.
4. **Build-tag-guarded files.** Check for `//go:build` tags before deleting a test helper — a symbol used only by a build-tagged file is still "used" under that build configuration. (`rg -n 'go:build'` — none observed, but re-verify.)
5. **Test helpers referenced only from `_test.go`.** U1000 treats test files as part of the package; a helper used only in tests is NOT dead. When grepping, include `*_test.go`.
6. **`cleanCrdVersion` / `scripts/clean-crd.go` preservation.** If `staticcheck`/`golangci-lint` reports `cleanCrdVersion` as unused (U1000), suppress it with `//nolint:staticcheck // kept for manual CRD cleanup` (or a targeted linter exclusion) rather than deleting it. Do not remove `scripts/clean-crd.go`.
7. **`emperror.dev/errors` / `go-funk` preservation.** Do not run a `go mod tidy` that would drop these (they stay as direct deps). No `//nolint` needed — they are not flagged as deprecated.
8. **No behavioral change.** All edits must be neutral with respect to runtime behavior, resource naming, labels/annotations, and CRD output.
9. **`gofmt` / `go vet` must stay clean** after every batch (§7).

---

## 6. Validation & Verification Plan

Run after each remediation batch (§7) and again at the very end:

```bash
# 1. Formatting / vet
gofmt -l .                                   # expect no output
go vet ./...                                 # expect clean

# 2. Compile
make build                                   # runs generate + fmt + vet + build manager binary
go build ./...                               # builds every package incl. scripts/hack

# 3. Static analysis — zero remaining SA1019 / U1000 (after applying §3.D exclusions)
staticcheck ./...                            # expect no SA1019, no U1000 (except suppressed cleanCrdVersion)
golangci-lint run --enable=unused,staticcheck,ineffassign,govet ./...  # if installed

# 4. Generated-code drift check (only if a generated file was implicated)
make generate && make manifests
git diff --exit-code -- 'api/**/zz_generated.deepcopy.go' 'config/crd/**' 'config/**/*.yaml'

# 5. Tests
make test                                     # envtest suites
# or, scoped and faster:
go test ./api/... ./internal/controller/... ./pkg/... -count=1
```

**Tests that may need updating because of this change:**
- The five restored tests (B1) may fail against current builder behavior — fix stale assertions only.
- No test imports `emperror.dev/errors`/`go-funk` need changing (those libraries stay).

`go mod tidy` is **not** part of this plan's cleanup (no dependency is being removed); run it only if a SA1019 fix incidentally drops an unused transitive import, and confirm it produces no unexpected drift.

---

## 7. Ordering & Risk Control

Execute in small batches, rebuilding/testing between batches. Do NOT delete a symbol until a repo-wide `rg -n '\bSymbol\b'` (excluding `zz_generated*`) returns zero references outside its declaration, and it is not subject to a §3.D exclusion.

1. **Baseline** — capture current state:
   ```bash
   staticcheck ./... > /tmp/staticcheck.before.txt   # capture SA1019 + U1000
   go build ./... && go vet ./...
   ```
2. **Dead-file restoration (B1)** — `git mv` the five `secret_ca_*_builder_test..go` → `secret_ca_*_builder_test.go`; run `go test ./internal/controller/{metricbeat,kibana,logstash,filebeat}/...`; fix stale test expectations only.
3. **Triage SA1019 (genuinely deprecated) usages** — for each `staticcheck -checks=SA1019` hit, read the `// Deprecated:` doc and apply the documented replacement (library-provided). `go build ./...` after each. Do not hand-roll equivalents.
4. **Triage U1000 (genuinely dead) symbols** — for each `staticcheck -checks=U1000` hit, confirm it is truly unused and not subject to a §3.D exclusion, then delete. `go build ./...` after each. Suppress (do not delete) `cleanCrdVersion` and any other deliberately-kept symbol.
5. **Regenerate generated code last** (if any generated file was implicated) — `make generate && make manifests`, verify no drift.
6. **Final validation** — run the full §6 sequence and confirm zero remaining SA1019/U1000 (after exclusions), clean `gofmt`, clean `go vet`, and `make test` green.

**Regression guardrails**
- Never change builder/reconciler naming, labels, annotations, or error strings.
- Never replace a library function with a local reimplementation; only apply the library's documented replacement.

---

## 8. Open Questions / Items Requiring Confirmation

1. **B1 dead test files — restore vs delete.** *(Recommended: restore via `git mv`, then fix any stale assertions.)* Confirm the team prefers restoring coverage over deleting the tests.
2. **`disaster37/*` SA1019 hits (if any)** — if `staticcheck` flags a deprecated symbol inside a `disaster37/*` helper, confirm the team wants the documented replacement applied (vs. pinning/ignoring). *(Recommended: apply the documented replacement.)*
3. **`ci/dagger` module boundary** — confirm whether `ci/dagger` is its own Go module (`dagger.json` / `dagger.gen.go`). If so, `staticcheck`/`gofmt`/`go vet` must be run separately in that directory, and the `cleanCrdVersion` suppression (if any) lives there.
4. **Exported-symbol policy in `api/**`** — any U1000 hit on an **exported** symbol in `api/**` should be treated as "needs human decision / keep" (quasi-public CRD surface), while unexported/internal symbols can be deleted freely. Confirm this policy.
