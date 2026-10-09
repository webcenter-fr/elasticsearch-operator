# es-handler: Add Rolling Restart Support Methods

**Repository**: `/projects/es-handler` (separate Go module: `github.com/disaster37/es-handler/v9`)
**Depends on**: `github.com/disaster37/elasticsearch/v9` (already has `Shutdown()`, `Nodes()`, `Cat()` services)

## Goal

Add methods to `ElasticsearchHandler` interface for:
1. **Transient cluster settings** (allocation/rebalance control) — safer than persistent
2. **Node Shutdown API** (ES >= 7.15.2) — graceful node restarts
3. **Shard listing** — for safety predicates
4. **Node info** — for pod name → node ID mapping
5. **Cluster info** — for version detection

## Current State

**File**: `cluster.go` — has persistent settings only:
```go
func (h *ElasticsearchHandlerImpl) EnableRoutingRebalance() error  // persistent: rebalance=all
func (h *ElasticsearchHandlerImpl) DisableRoutingRebalance() error // persistent: rebalance=none
func (h *ElasticsearchHandlerImpl) EnableRoutingAllocation() error  // persistent: allocation=all
func (h *ElasticsearchHandlerImpl) DisableRoutingAllocation() error // persistent: allocation=primaries
```

**Problem**: Persistent settings survive cluster restarts. If operator crashes mid-upgrade, cluster stuck with disabled rebalancing/allocation.

## Changes

### 1. Add Transient Settings Methods

**File**: `cluster.go` (add to existing file)

```go
package eshandler

import (
	"context"
)

// --- Transient allocation/rebalance settings ---

// setRoutingSettingTransient persists a transient cluster setting with the given value.
func (h *ElasticsearchHandlerImpl) setRoutingSettingTransient(key, value string) error {
	settings := map[string]interface{}{
		"transient": map[string]interface{}{
			key: value,
		},
	}
	_, err := h.client.Cluster().PutSettings(context.Background(), settings, nil)
	return err
}

// DisableReplicaShardsAllocationTransient sets cluster.routing.allocation.enable=primaries (transient)
// Only primary shards can be allocated; replica shards are not allocated.
// Use during rolling restart to avoid replica replication to other nodes.
func (h *ElasticsearchHandlerImpl) DisableReplicaShardsAllocationTransient() error {
	return h.setRoutingSettingTransient("cluster.routing.allocation.enable", "primaries")
}

// EnableShardAllocationTransient sets cluster.routing.allocation.enable=all (transient)
// Re-enables full shard allocation after rolling restart.
func (h *ElasticsearchHandlerImpl) EnableShardAllocationTransient() error {
	return h.setRoutingSettingTransient("cluster.routing.allocation.enable", "all")
}

// DisableRoutingRebalanceTransient sets cluster.routing.rebalance.enable=none (transient)
// Prevents shard rebalancing during rolling restart (avoids unnecessary shard movement).
func (h *ElasticsearchHandlerImpl) DisableRoutingRebalanceTransient() error {
	return h.setRoutingSettingTransient("cluster.routing.rebalance.enable", "none")
}

// EnableRoutingRebalanceTransient sets cluster.routing.rebalance.enable=all (transient)
// Re-enables shard rebalancing after rolling restart.
func (h *ElasticsearchHandlerImpl) EnableRoutingRebalanceTransient() error {
	return h.setRoutingSettingTransient("cluster.routing.rebalance.enable", "all")
}

// RemoveTransientAllocationSettings removes transient allocation and rebalance settings.
// Use to clean up after rolling restart or when migrating to Node Shutdown API.
func (h *ElasticsearchHandlerImpl) RemoveTransientAllocationSettings() error {
	settings := map[string]interface{}{
		"transient": map[string]interface{}{
			"cluster.routing.allocation.enable": nil,
			"cluster.routing.rebalance.enable":  nil,
		},
	}
	_, err := h.client.Cluster().PutSettings(context.Background(), settings, nil)
	return err
}

// GetClusterRoutingAllocation retrieves current transient allocation settings.
func (h *ElasticsearchHandlerImpl) GetClusterRoutingAllocation() (allocationEnable, rebalanceEnable string, err error) {
	resp, err := h.client.Cluster().GetSettings(context.Background(), nil)
	if err != nil {
		return "", "", err
	}
	// Parse transient settings from response
	// Response structure: {"transient": {"cluster": {"routing": {"allocation": {"enable": "..."}, "rebalance": {"enable": "..."}}}}}
	// Implementation depends on esapi.ClusterGetSettingsResponse structure
	if transient, ok := resp.Transient["cluster"].(map[string]interface{}); ok {
		if routing, ok := transient["routing"].(map[string]interface{}); ok {
			if alloc, ok := routing["allocation"].(map[string]interface{}); ok {
				if enable, ok := alloc["enable"].(string); ok {
					allocationEnable = enable
				}
			}
			if rebal, ok := routing["rebalance"].(map[string]interface{}); ok {
				if enable, ok := rebal["enable"].(string); ok {
					rebalanceEnable = enable
				}
			}
		}
	}
	return allocationEnable, rebalanceEnable, nil
}
```

### 2. Add Node Shutdown API Methods

**File**: `shutdown.go` (new file)

```go
package eshandler

import (
	"context"
	"fmt"

	esapi "github.com/disaster37/elasticsearch/v9/api"
)

// ShutdownType represents the type of node shutdown.
type ShutdownType string

const (
	// ShutdownTypeRestart indicates a temporary shutdown (node will return).
	ShutdownTypeRestart ShutdownType = "restart"
	// ShutdownTypeRemove indicates a permanent shutdown (node will not return).
	ShutdownTypeRemove ShutdownType = "remove"
)

// ShutdownStatus represents the status of a node shutdown.
type ShutdownStatus string

const (
	ShutdownNotStarted ShutdownStatus = "NOT_STARTED"
	ShutdownInProgress ShutdownStatus = "IN_PROGRESS"
	ShutdownComplete   ShutdownStatus = "COMPLETE"
	ShutdownStalled    ShutdownStatus = "STALLED"
)

// NodeShutdownInfo contains information about a node shutdown.
type NodeShutdownInfo struct {
	NodeID            string
	Type              string
	Reason            string
	Status            ShutdownStatus
	AllocationDelay   string
	ShutdownStarted   string
	ShutdownCompleted string
}

// PutNodeShutdown requests a node shutdown.
// Available since Elasticsearch 7.15.2.
// shutdownType: "restart" (temporary) or "remove" (permanent)
// reason: arbitrary string for tracking (e.g., "rolling-restart", resource version)
func (h *ElasticsearchHandlerImpl) PutNodeShutdown(ctx context.Context, nodeID string, shutdownType ShutdownType, reason string) error {
	body := map[string]interface{}{
		"type":   string(shutdownType),
		"reason": reason,
	}
	_, err := h.client.Shutdown().PutNode(ctx, nodeID, body)
	return err
}

// GetNodeShutdown retrieves the shutdown status for a specific node.
// Returns nil if no shutdown is in progress for the node.
func (h *ElasticsearchHandlerImpl) GetNodeShutdown(ctx context.Context, nodeID string) (*NodeShutdownInfo, error) {
	resp, err := h.client.Shutdown().GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Nodes) == 0 {
		return nil, nil
	}
	node := resp.Nodes[0]
	info := &NodeShutdownInfo{
		NodeID:          node.NodeID,
		Type:            node.Type,
		Reason:          node.Reason,
		AllocationDelay: node.AllocationDelay,
		ShutdownStarted: node.ShutdownStarted,
	}
	// Parse status from map
	if statusMap, ok := node.Status["shard_migration"].(map[string]interface{}); ok {
		if status, ok := statusMap["status"].(string); ok {
			info.Status = ShutdownStatus(status)
		}
	}
	return info, nil
}

// GetAllNodeShutdowns retrieves all ongoing node shutdowns.
func (h *ElasticsearchHandlerImpl) GetAllNodeShutdowns(ctx context.Context) ([]NodeShutdownInfo, error) {
	resp, err := h.client.Shutdown().GetNode(ctx, "")
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	shutdowns := make([]NodeShutdownInfo, 0, len(resp.Nodes))
	for _, node := range resp.Nodes {
		info := NodeShutdownInfo{
			NodeID:          node.NodeID,
			Type:            node.Type,
			Reason:          node.Reason,
			AllocationDelay: node.AllocationDelay,
			ShutdownStarted: node.ShutdownStarted,
		}
		if statusMap, ok := node.Status["shard_migration"].(map[string]interface{}); ok {
			if status, ok := statusMap["status"].(string); ok {
				info.Status = ShutdownStatus(status)
			}
		}
		shutdowns = append(shutdowns, info)
	}
	return shutdowns, nil
}

// DeleteNodeShutdown cancels or cleans up a node shutdown.
func (h *ElasticsearchHandlerImpl) DeleteNodeShutdown(ctx context.Context, nodeID string) error {
	_, err := h.client.Shutdown().DeleteNode(ctx, nodeID)
	return err
}
```

### 3. Add Node Info Methods

**File**: `nodes.go` (new file)

```go
package eshandler

import (
	"context"

	esapi "github.com/disaster37/elasticsearch/v9/api"
)

// NodeInfo contains basic information about an Elasticsearch node.
type NodeInfo struct {
	ID       string
	Name     string
	Version  string
	Roles    []string
	Host     string
	IP       string
}

// GetNodesInfo retrieves information about all nodes in the cluster.
// Returns a map of node name -> NodeInfo.
func (h *ElasticsearchHandlerImpl) GetNodesInfo(ctx context.Context) (map[string]NodeInfo, error) {
	resp, err := h.client.Nodes().Info(ctx, &esapi.NodesInfoRequest{})
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]NodeInfo, len(resp.Nodes))
	for id, node := range resp.Nodes {
		nodes[node.Name] = NodeInfo{
			ID:      id,
			Name:    node.Name,
			Version: node.Version,
			Roles:   node.Roles,
			Host:    node.Host,
			IP:      node.IP,
		}
	}
	return nodes, nil
}

// GetNodeNameToIDMap returns a map of node name -> node ID.
// Useful for Node Shutdown API (which requires node ID, not name).
func (h *ElasticsearchHandlerImpl) GetNodeNameToIDMap(ctx context.Context) (map[string]string, error) {
	nodes, err := h.GetNodesInfo(ctx)
	if err != nil {
		return nil, err
	}
	nameToID := make(map[string]string, len(nodes))
	for name, node := range nodes {
		nameToID[name] = node.ID
	}
	return nameToID, nil
}

// IsNodeInCluster checks if a node (by name) is currently in the cluster.
func (h *ElasticsearchHandlerImpl) IsNodeInCluster(ctx context.Context, nodeName string) (bool, error) {
	nodes, err := h.GetNodesInfo(ctx)
	if err != nil {
		return false, err
	}
	_, exists := nodes[nodeName]
	return exists, nil
}
```

### 4. Add Shard Listing Methods

**File**: `shard.go` (new file)

```go
package eshandler

import (
	"context"
	"fmt"

	esapi "github.com/disaster37/elasticsearch/v9/api"
)

// ShardInfo represents information about a single shard.
type ShardInfo struct {
	Index    string
	Shard    string
	Primary  bool   // true = primary, false = replica
	State    string // STARTED, UNASSIGNED, INITIALIZING, RELOCATING
	NodeName string // empty if unassigned
	NodeID   string // empty if unassigned
}

// Key returns a unique key for the shard (index-shard).
func (s ShardInfo) Key() string {
	return fmt.Sprintf("%s-%s", s.Index, s.Shard)
}

// IsPrimary returns true if this is a primary shard.
func (s ShardInfo) IsPrimary() bool {
	return s.Primary
}

// GetShards retrieves all shards in the cluster using the Cat API.
func (h *ElasticsearchHandlerImpl) GetShards(ctx context.Context) ([]ShardInfo, error) {
	resp, err := h.client.Cat().Shards(ctx, &esapi.CatShardsParams{
		Format: "json",
	})
	if err != nil {
		return nil, err
	}
	shards := make([]ShardInfo, 0, len(resp))
	for _, s := range resp {
		shards = append(shards, ShardInfo{
			Index:    s.Index,
			Shard:    s.Shard,
			Primary:  s.Prirep == "p",
			State:    s.State,
			NodeName: s.Node,
			NodeID:   s.NodeId,
		})
	}
	return shards, nil
}

// GetShardsByNode returns shards grouped by node name.
// Unassigned shards (NodeName == "") are not included in any group.
func (h *ElasticsearchHandlerImpl) GetShardsByNode(ctx context.Context) (map[string][]ShardInfo, error) {
	shards, err := h.GetShards(ctx)
	if err != nil {
		return nil, err
	}
	byNode := make(map[string][]ShardInfo)
	for _, shard := range shards {
		if shard.NodeName != "" {
			byNode[shard.NodeName] = append(byNode[shard.NodeName], shard)
		}
	}
	return byNode, nil
}

// GetShardsByIndex returns shards grouped by index name.
func (h *ElasticsearchHandlerImpl) GetShardsByIndex(ctx context.Context) (map[string][]ShardInfo, error) {
	shards, err := h.GetShards(ctx)
	if err != nil {
		return nil, err
	}
	byIndex := make(map[string][]ShardInfo)
	for _, shard := range shards {
		byIndex[shard.Index] = append(byIndex[shard.Index], shard)
	}
	return byIndex, nil
}

// HasShardActivity checks if there are any initializing or relocating shards.
// Uses cluster health with wait_for_events=languid for accuracy.
func (h *ElasticsearchHandlerImpl) HasShardActivity(ctx context.Context) (bool, error) {
	health, err := h.client.Cluster().Health(ctx, nil, &esapi.ClusterHealthParams{
		WaitForEvents: "languid",
		Timeout:       "0s",
	})
	if err != nil {
		return false, err
	}
	// Check for initializing or relocating shards
	return health.InitializingShards > 0 || health.RelocatingShards > 0, nil
}

// CountStartedReplicas counts STARTED replicas for a given shard key (index-shard).
// Excludes shards on the specified node (if excludeNode is not empty).
func (h *ElasticsearchHandlerImpl) CountStartedReplicas(ctx context.Context, index, shard, excludeNode string) (total, started int, err error) {
	shards, err := h.GetShards(ctx)
	if err != nil {
		return 0, 0, err
	}
	shardKey := fmt.Sprintf("%s-%s", index, shard)
	for _, s := range shards {
		if s.Key() != shardKey {
			continue
		}
		if s.Primary {
			continue // Skip primaries
		}
		if excludeNode != "" && s.NodeName == excludeNode {
			continue // Skip shards on excluded node
		}
		total++
		if s.State == "STARTED" {
			started++
		}
	}
	return total, started, nil
}
```

### 5. Add Cluster Info Method

**File**: `cluster.go` (add to existing file)

```go
// ClusterInfo contains basic cluster information.
type ClusterInfo struct {
	Name        string
	Version     string
	Tagline     string
}

// GetClusterInfo retrieves basic cluster information including version.
func (h *ElasticsearchHandlerImpl) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	resp, err := h.client.Info().Get(ctx)
	if err != nil {
		return nil, err
	}
	return &ClusterInfo{
		Name:    resp.ClusterName,
		Version: resp.Version.Number,
		Tagline: resp.Tagline,
	}, nil
}

// GetVersion returns just the Elasticsearch version string.
func (h *ElasticsearchHandlerImpl) GetVersion(ctx context.Context) (string, error) {
	info, err := h.GetClusterInfo(ctx)
	if err != nil {
		return "", err
	}
	return info.Version, nil
}
```

### 6. Add Flush Method

**File**: `cluster.go` (add to existing file)

```go
// Flush requests a flush on all indices.
// Optimizes index recovery when nodes restart.
func (h *ElasticsearchHandlerImpl) Flush(ctx context.Context) error {
	// Use the Indices service flush API
	_, err := h.client.Indices().Flush(ctx, nil, nil)
	return err
}

// SyncedFlush requests a synced flush (deprecated in ES 8.0, removed in 9.0).
// Best-effort: returns nil on 409 CONFLICT.
func (h *ElasticsearchHandlerImpl) SyncedFlush(ctx context.Context) error {
	// Implementation depends on available API
	// For ES < 8.0: POST /_flush/synced
	// For ES >= 8.0: use regular Flush
	return h.Flush(ctx)
}
```

### 7. Update Interface

**File**: `elasticsearch.go` — add to `ElasticsearchHandler` interface:

```go
// Cluster scope (existing + new)
ClusterHealth() (health *esapi.ClusterHealthResponse, err error)
EnableRoutingRebalance() (err error)
DisableRoutingRebalance() (err error)
EnableRoutingAllocation() (err error)
DisableRoutingAllocation() (err error)

// Transient settings (new)
DisableReplicaShardsAllocationTransient() error
EnableShardAllocationTransient() error
DisableRoutingRebalanceTransient() error
EnableRoutingRebalanceTransient() error
RemoveTransientAllocationSettings() error
GetClusterRoutingAllocation() (allocationEnable, rebalanceEnable string, err error)

// Cluster info (new)
GetClusterInfo(ctx context.Context) (*ClusterInfo, error)
GetVersion(ctx context.Context) (string, error)

// Flush (new)
Flush(ctx context.Context) error
SyncedFlush(ctx context.Context) error

// Node shutdown scope (new)
PutNodeShutdown(ctx context.Context, nodeID string, shutdownType ShutdownType, reason string) error
GetNodeShutdown(ctx context.Context, nodeID string) (*NodeShutdownInfo, error)
GetAllNodeShutdowns(ctx context.Context) ([]NodeShutdownInfo, error)
DeleteNodeShutdown(ctx context.Context, nodeID string) error

// Node info scope (new)
GetNodesInfo(ctx context.Context) (map[string]NodeInfo, error)
GetNodeNameToIDMap(ctx context.Context) (map[string]string, error)
IsNodeInCluster(ctx context.Context, nodeName string) (bool, error)

// Shard scope (new)
GetShards(ctx context.Context) ([]ShardInfo, error)
GetShardsByNode(ctx context.Context) (map[string][]ShardInfo, error)
GetShardsByIndex(ctx context.Context) (map[string][]ShardInfo, error)
HasShardActivity(ctx context.Context) (bool, error)
CountStartedReplicas(ctx context.Context, index, shard, excludeNode string) (total, started int, err error)
```

### 8. Regenerate Mocks

```bash
cd /projects/es-handler
mockgen --build_flags=--mod=mod -destination=mocks/elasticsearch_handler.go -package=mocks github.com/disaster37/es-handler/v9 ElasticsearchHandler
```

Or manually add mock methods to `mocks/elasticsearch_handler.go`.

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `cluster.go` | Modify | Add transient settings, cluster info, flush methods |
| `shutdown.go` | Create | Node Shutdown API methods |
| `nodes.go` | Create | Node info methods |
| `shard.go` | Create | Shard listing methods |
| `elasticsearch.go` | Modify | Update interface with new methods |
| `mocks/elasticsearch_handler.go` | Regenerate | Update mocks |
| `cluster_test.go` | Modify | Add tests for new methods |
| `shutdown_test.go` | Create | Tests for shutdown methods |
| `shard_test.go` | Create | Tests for shard methods |
| `nodes_test.go` | Create | Tests for node methods |

## Testing

```bash
cd /projects/es-handler

# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test -v ./... -run TestDisableReplicaShardsAllocationTransient
```

## Dependencies

- `github.com/disaster37/elasticsearch/v9` — already a dependency, provides:
  - `client.Shutdown()` → `api.ShutdownService` (GetNode, PutNode, DeleteNode)
  - `client.Nodes()` → `api.NodesService` (Info, Stats)
  - `client.Cat()` → `api.CatService` (Shards, Nodes)
  - `client.Cluster()` → `api.ClusterService` (Health, GetSettings, PutSettings)
  - `client.Info()` → `api.InfoService` (Get)
  - `client.Indices()` → `api.IndicesService` (Flush)

## Version Compatibility

| Feature | Min ES Version | Notes |
|---------|---------------|-------|
| Transient settings | All | Core ES feature |
| Node Shutdown API | 7.15.2 | `PUT/GET/DELETE /_nodes/{id}/shutdown` |
| Cat Shards API | All | `GET /_cat/shards` |
| Flush API | All | `POST /_flush` |
| Synced Flush | < 8.0 | Deprecated in 8.0, removed in 9.0 |
