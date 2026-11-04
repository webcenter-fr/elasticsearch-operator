package v1

import (
	"github.com/disaster37/operator-sdk-extra/v3/pkg/object"
	localobject "github.com/webcenter-fr/elasticsearch-operator/pkg/object"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetInternalName permit to get the logical name of the discover
func (h Kafka) GetInternalName() string {
	if h.Spec.Name != nil && *h.Spec.Name != "" {
		return *h.Spec.Name
	}

	return h.Name
}

// GetStatus implement the object.MultiPhaseObject
func (h *Kafka) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// GetDiscoverStatus implement the discover.DiscoverObject
func (h *Kafka) GetDiscoverStatus() localobject.DiscoverObjectStatus {
	return &h.Status
}

// IsManaged permit to know if Kafka is managed by operator
func (h KafkaRef) IsManaged() bool {
	return h.ManagedKafkaRef != nil && h.ManagedKafkaRef.Name != ""
}

// IsExternal permit to know if Kafka is external (not managed by operator)
func (h KafkaRef) IsExternal() bool {
	return h.ExternalKafkaRef != nil && len(h.ExternalKafkaRef.Addresses) > 0
}

// ValidateField permit to validate field from webhook
func (h KafkaRef) ValidateField() *field.Error {
	// Check we provide Kafka cluster
	if !h.IsExternal() && !h.IsManaged() {
		return field.Required(field.NewPath("spec").Child("kafkaRef"), "You need to provide managed or external Kafka cluster")
	}

	// Check not provide both
	if h.IsExternal() && h.IsManaged() {
		return field.Invalid(field.NewPath("spec").Child("kafkaRef"), h, "You can't provide both managed and external Kafka cluster")
	}

	return nil
}
