package v1

import (
	"github.com/disaster37/operator-sdk-extra/v3/pkg/object"
	localobject "github.com/webcenter-fr/elasticsearch-operator/pkg/object"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetInternalName permit to get the logical name of the discover
func (h Logstash) GetInternalName() string {
	if h.Spec.Name != nil && *h.Spec.Name != "" {
		return *h.Spec.Name
	}

	return h.Name
}

// GetStatus implement the object.MultiPhaseObject
func (h *Logstash) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// GetDiscoverStatus implement the discover.DiscoverObject
func (h *Logstash) GetDiscoverStatus() localobject.DiscoverObjectStatus {
	return &h.Status
}

// IsManaged permit to know if Logstash is managed by operator
func (h LogstashRef) IsManaged() bool {
	return h.ManagedLogstashRef != nil && h.ManagedLogstashRef.Name != ""
}

// IsExternal permit to know if Logstash is external (not managed by operator)
func (h LogstashRef) IsExternal() bool {
	return h.ExternalLogstashRef != nil && len(h.ExternalLogstashRef.Addresses) > 0
}

// ValidateField permit to validate field from webhook
func (h LogstashRef) ValidateField() *field.Error {
	// Check we provide Logstash
	if !h.IsExternal() && !h.IsManaged() {
		return field.Required(field.NewPath("spec").Child("logstashRef"), "You need to provide managed or external Logstash")
	}

	if h.IsExternal() && h.IsManaged() {
		return field.Invalid(field.NewPath("spec").Child("logstashRef"), "", "You can't provide managed and external Logstash at the same time")
	}

	return nil
}
