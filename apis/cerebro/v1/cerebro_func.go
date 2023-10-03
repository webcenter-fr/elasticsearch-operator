package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Cerebro) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsIngressEnabled return true if ingress is enabled
func (h *Cerebro) IsIngressEnabled() bool {
	if h.Spec.Endpoint.Ingress != nil && h.Spec.Endpoint.Ingress.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h *Cerebro) IsLoadBalancerEnabled() bool {
	if h.Spec.Endpoint.LoadBalancer != nil && h.Spec.Endpoint.LoadBalancer.Enabled {
		return true
	}

	return false
}
