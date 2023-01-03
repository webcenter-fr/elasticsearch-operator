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

// IsElasticsearchRef return true if ElasticsearchRef is setted
func (h *Kibana) IsElasticsearchRef() bool {
	if h.Spec.ElasticsearchRef != nil && h.Spec.ElasticsearchRef.Name != "" {
		return true
	}

	return false
}
