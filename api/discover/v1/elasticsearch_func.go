package v1

import (
	"github.com/disaster37/operator-sdk-extra/v3/pkg/object"
	localobject "github.com/webcenter-fr/elasticsearch-operator/pkg/object"
)

// GetInternalName permit to get the logical name of the discover
func (h Elasticsearch) GetInternalName() string {
	if h.Spec.Name != nil && *h.Spec.Name != "" {
		return *h.Spec.Name
	}

	return h.Name
}

// GetStatus implement the object.MultiPhaseObject
func (h *Elasticsearch) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// GetDiscoverStatus implement the discover.DiscoverObject
func (h *Elasticsearch) GetDiscoverStatus() localobject.DiscoverObjectStatus {
	return &h.Status
}
