package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Kibana) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

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

// IsPdb return true if PDB is enabled
func (h *Kibana) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}
