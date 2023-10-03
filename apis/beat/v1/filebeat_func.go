package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Filebeat) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Filebeat) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsPersistence return true if persistence is enabled
func (h *Filebeat) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaimSpec != nil) {
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

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *Filebeat) IsMetricbeatMonitoring() bool {

	if h.Spec.Monitoring.Metricbeat != nil && h.Spec.Monitoring.Metricbeat.Enabled && h.Spec.Deployment.Replicas > 0 {
		return true
	}

	return false
}

// IsPdb return true if PDB is enabled
func (h *Filebeat) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}
