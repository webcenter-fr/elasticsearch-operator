package v1alpha1

// IsSelfManagedSecretForTls return true if the operator manage the certificates for TLS
// It return false if secret is provided
func (h *Kibana) IsSelfManagedSecretForTls() bool {
	return h.Spec.Tls.CertificateSecretRef == nil
}

// IsTlsEnabled return true if TLS is enabled to access on Kibana
func (h *Kibana) IsTlsEnabled() bool {
	if h.Spec.Tls.Enabled != nil && !*h.Spec.Tls.Enabled {
		return false
	}
	return true
}

// IsIngressEnabled return true if ingress is enabled
func (h *Kibana) IsIngressEnabled() bool {
	if h.Spec.Endpoint.Ingress != nil && h.Spec.Endpoint.Ingress.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h *Kibana) IsLoadBalancerEnabled() bool {
	if h.Spec.Endpoint.LoadBalancer != nil && h.Spec.Endpoint.LoadBalancer.Enabled {
		return true
	}

	return false
}

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Kibana) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}
