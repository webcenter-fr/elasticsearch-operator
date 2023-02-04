package v1alpha1

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Logstash) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsOneForEachLogstashInstance return true if need one ingress for each logstash instance
func (h *Ingress) IsOneForEachLogstashInstance() bool {
	if h.OneForEachLogstashInstance != nil && *h.OneForEachLogstashInstance {
		return true
	}

	return false
}

// IsPersistence return true if persistence is enabled
func (h *Logstash) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaimSpec != nil) {
		return true
	}

	return false
}
