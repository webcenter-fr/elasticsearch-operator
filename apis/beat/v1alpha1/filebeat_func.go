package v1alpha1

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
func (h LogstashRef) IsManaged() bool {
	return h.ManagedLogstashRef != nil && h.ManagedLogstashRef.Name != ""
}

// IsExternal permit to know if Logstash is external (not managed by operator)
func (h LogstashRef) IsExternal() bool {
	return h.ExternalLogstashRef != nil && len(h.ExternalLogstashRef.Addresses) > 0
}
