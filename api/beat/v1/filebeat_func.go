package v1

import (
	"github.com/disaster37/operator-sdk-extra/v2/pkg/object"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Filebeat) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsPersistence return true if persistence is enabled
func (h *Filebeat) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaim != nil) {
		return true
	}

	return false
}

// IsManaged permit to know if Logstash is managed by operator
func (h FilebeatLogstashRef) IsManaged() bool {
	return h.ManagedLogstashRef != nil && h.ManagedLogstashRef.Name != ""
}

// IsExternal permit to know if Logstash is external (not managed by operator)
func (h FilebeatLogstashRef) IsExternal() bool {
	return h.ExternalLogstashRef != nil && len(h.ExternalLogstashRef.Addresses) > 0
}

// ValidateField permit to validate field from webhook
func (h FilebeatLogstashRef) ValidateField() *field.Error {
	// Check we provide Opensearch cluster
	if !h.IsExternal() && !h.IsManaged() {
		return field.Required(field.NewPath("spec").Child("LogstashRef"), "You need to provide managed or external Logstash target")
	}

	return nil
}

// IsPdb return true if PDB is enabled
func (h *Filebeat) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}

// IsEnabled return true if PKI is enabled
func (h FilebeatPkiSpec) IsEnabled() bool {
	if h.Enabled == nil || *h.Enabled {
		return true
	}
	return false
}
