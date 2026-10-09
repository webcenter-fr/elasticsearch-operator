# Replace Local `DefaultControllerRateLimiter` with operator-sdk-extra Version

## Goal

Remove the duplicate local `DefaultControllerRateLimiter()` function from `internal/controller/common/controller.go` and use the identical implementation from `github.com/disaster37/operator-sdk-extra/v3/pkg/controller` instead.

## Context

### Current State

The project has **two identical** rate limiter implementations:

| Location | Function | Type |
|----------|----------|------|
| `internal/controller/common/controller.go:99` | `DefaultControllerRateLimiter()` | Local, hardcoded to `reconcile.Request` |
| `operator-sdk-extra/v3/pkg/controller/helper.go:115` | `DefaultControllerRateLimiter[T comparable]()` | Library, generic |

Both implementations are **byte-for-byte identical** in behavior:

```go
// Both use:
workqueue.NewTypedMaxOfRateLimiter(
    workqueue.NewTypedItemExponentialFailureRateLimiter(1*time.Second, 1000*time.Second),
    &workqueue.TypedBucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
)
```

### Current Usage

| Controllers | Count | Using |
|-------------|-------|-------|
| kibanaapi (logstashpipeline, role, userspace), metricbeat | 4 | `controller.DefaultControllerRateLimiter[reconcile.Request]()` (library) |
| elasticsearch, kibana, logstash, filebeat, cerebro, elasticsearchapi (10 controllers) | 15 | `common.DefaultControllerRateLimiter()` (local) |

### Files Using Local Implementation (15 files)

```
internal/controller/elasticsearch/elasticsearch_controller.go:194
internal/controller/kibana/kibana_controller.go:179
internal/controller/logstash/logstash_controller.go:174
internal/controller/filebeat/filebeat_controller.go:167
internal/controller/cerebro/cerebro_controller.go:149
internal/controller/elasticsearchapi/watch_controller.go:96
internal/controller/elasticsearchapi/rolemapping_controller.go:97
internal/controller/elasticsearchapi/componenttemplate_controller.go:97
internal/controller/elasticsearchapi/user_controller.go:103
internal/controller/elasticsearchapi/role_controller.go:97
internal/controller/elasticsearchapi/indexlifecyclepolicy_controller.go:97
internal/controller/elasticsearchapi/license_controller.go:104
internal/controller/elasticsearchapi/snapshotlifecyclepolicy_controller.go:96
internal/controller/elasticsearchapi/snapshotrepository_controller.go:97
internal/controller/elasticsearchapi/indextemplate_controller.go:97
```

## Target State

All 19 controllers use `controller.DefaultControllerRateLimiter[reconcile.Request]()` from operator-sdk-extra.

The local `DefaultControllerRateLimiter()` function is **deleted** from `internal/controller/common/controller.go`.

## Implementation Steps

### Step 1: Update Imports in 15 Controller Files

For each of the 15 files listed above:

1. Add import: `"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"`
   - Note: Some files may already import this package (check first)
   - If the file already imports `controller` from operator-sdk-extra, skip this step

2. Remove import: `"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"`
   - **Only if** `common` is not used for other functions in that file
   - Check for other uses of `common.` in the file before removing

### Step 2: Replace Rate Limiter Calls

In each of the 15 files, replace:

```go
RateLimiter: common.DefaultControllerRateLimiter(),
```

with:

```go
RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
```

### Step 3: Clean Up `internal/controller/common/controller.go`

Remove the `DefaultControllerRateLimiter()` function (lines 99-104):

```go
func DefaultControllerRateLimiter() workqueue.TypedRateLimiter[reconcile.Request] {
	return workqueue.NewTypedMaxOfRateLimiter[reconcile.Request](
		workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](1*time.Second, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}
```

Remove now-unused imports from `internal/controller/common/controller.go`:
- `"golang.org/x/time/rate"`
- `"k8s.io/client-go/util/workqueue"`
- `"sigs.k8s.io/controller-runtime/pkg/reconcile"`

### Step 4: Verify Build

```bash
go build ./...
go vet ./...
```

### Step 5: Run Tests

```bash
go test ./internal/controller/... -count=1
```

## Detailed File Changes

### Files to Modify (15 controller files)

Each file needs:
1. Add `"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"` import (if not present)
2. Replace `common.DefaultControllerRateLimiter()` with `controller.DefaultControllerRateLimiter[reconcile.Request]()`
3. Remove `"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"` import (if no other uses)

### File to Clean Up (1 file)

`internal/controller/common/controller.go`:
- Remove `DefaultControllerRateLimiter()` function
- Remove unused imports: `golang.org/x/time/rate`, `k8s.io/client-go/util/workqueue`, `sigs.k8s.io/controller-runtime/pkg/reconcile`

## Validation

### Build Validation

```bash
# Build all packages
go build ./...

# Vet all packages
go vet ./...

# Check for unused imports
goimports -l internal/controller/
```

### Test Validation

```bash
# Run controller tests
go test ./internal/controller/... -count=1

# Run common package tests
go test ./internal/controller/common/... -count=1
```

### Grep Validation

Verify no remaining references to local rate limiter:

```bash
# Should return 0 results
grep -r "common.DefaultControllerRateLimiter" --include="*.go" .

# Should return 19 results (all controllers)
grep -r "controller.DefaultControllerRateLimiter\[reconcile.Request\]()" --include="*.go" .
```

## Risks

| Risk | Mitigation |
|------|------------|
| Import name collision (`controller` already used for something else) | Check each file's imports; use alias if needed (e.g., `sdkcontroller "github.com/disaster37/operator-sdk-extra/v3/pkg/controller"`) |
| `common` package still needed for other functions | Check each file for other `common.` usages before removing import |
| Breaking change in rate limiter behavior | None - implementations are identical |
| Test failures | Run full test suite; rate limiter behavior is unchanged |

## Rollback Plan

If issues arise:

1. Revert the changes to the 15 controller files
2. Restore the `DefaultControllerRateLimiter()` function in `internal/controller/common/controller.go`
3. Restore the removed imports

## Open Questions

None - the implementations are identical, so this is a safe refactoring.

## References

- operator-sdk-extra v3: `pkg/controller/helper.go:115` - `DefaultControllerRateLimiter[T comparable]()`
- Local implementation: `internal/controller/common/controller.go:99` - `DefaultControllerRateLimiter()`
- Already migrated controllers: `internal/controller/kibanaapi/*_controller.go`, `internal/controller/metricbeat/metricbeat_controller.go`
