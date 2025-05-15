package v1

import (
	"github.com/disaster37/operator-sdk-extra/v2/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Kibana) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsPdb return true if PDB is enabled
func (h *Kibana) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}
