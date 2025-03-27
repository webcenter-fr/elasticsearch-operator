package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Logstash) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsPersistence return true if persistence is enabled
func (h *Logstash) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaim != nil) {
		return true
	}

	return false
}

// IsPdb return true if PDB is enabled
func (h *Logstash) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}

// isEnabled return true if PKI is enabled
func (h LogstashPkiSpec) IsEnabled() bool {
	if h.Enabled == nil || *h.Enabled {
		return true
	}
	return false
}

// HasBeatCertificate return true if PKI use to generate beat certificates
func (h LogstashPkiSpec) HasBeatCertificate() bool {
	for _, data := range h.Tls {
		if data.Consumer == "beat" {
			return true
		}
	}

	return false
}
