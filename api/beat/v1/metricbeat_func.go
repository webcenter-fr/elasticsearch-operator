package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Metricbeat) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsPersistence return true if persistence is enabled
func (h *Metricbeat) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaim != nil) {
		return true
	}

	return false
}

// IsPdb return true if PDB is enabled
func (h *Metricbeat) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}
