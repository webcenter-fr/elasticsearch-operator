package v1

import (
	"github.com/disaster37/operator-sdk-extra/v2/pkg/object"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

// ValidateField permit to validate field from webhook
func (h ElasticsearchRef) ValidateField() *field.Error {
	// Check we provide Opensearch cluster
	if !h.IsExternal() && !h.IsManaged() {
		return field.Required(field.NewPath("spec").Child("elasticsearchRef"), "You need to provide managed or external Elasticsearch cluster")
	}

	return nil
}
