package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Host) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsManaged permit to know if Elasticsearch is managed by operator
func (h ElasticsearchRef) IsManaged() bool {
	return h.ManagedElasticsearchRef != nil && h.ManagedElasticsearchRef.Name != ""
}

// IsExternal permit to know if Elasticsearch is external (not managed by operator)
func (h ElasticsearchRef) IsExternal() bool {
	return h.ExternalElasticsearchRef != nil && h.ExternalElasticsearchRef.Address != "" && h.ExternalElasticsearchRef.Name != ""
}
