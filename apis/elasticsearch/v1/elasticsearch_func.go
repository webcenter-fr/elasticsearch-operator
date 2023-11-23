package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Elasticsearch) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsIngressEnabled return true if ingress is enabled
func (h *Elasticsearch) IsIngressEnabled() bool {
	if h.Spec.Endpoint.Ingress != nil && h.Spec.Endpoint.Ingress.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h *Elasticsearch) IsLoadBalancerEnabled() bool {
	if h.Spec.Endpoint.LoadBalancer != nil && h.Spec.Endpoint.LoadBalancer.Enabled {
		return true
	}

	return false
}

// IsSetVMMaxMapCount return true if SetVMMaxMapCount is enabled
func (h *Elasticsearch) IsSetVMMaxMapCount() bool {
	if h.Spec.SetVMMaxMapCount != nil && !*h.Spec.SetVMMaxMapCount {
		return false
	}

	return true
}

// IsPersistence return true if persistence is enabled
func (h ElasticsearchNodeGroupSpec) IsPersistence() bool {
	if h.Persistence != nil && (h.Persistence.Volume != nil || h.Persistence.VolumeClaimSpec != nil) {
		return true
	}

	return false
}

// IsPdb return true if PDB is enabled
func (h *Elasticsearch) IsPdb(nodeGroup ElasticsearchNodeGroupSpec) bool {
	if h.Spec.GlobalNodeGroup.PodDisruptionBudgetSpec != nil || nodeGroup.PodDisruptionBudgetSpec != nil || nodeGroup.Replicas > 1 {
		return true
	}

	return false
}

// IsBoostraping return true if cluster is already bootstraped
func (h *Elasticsearch) IsBoostrapping() bool {
	if h.Status.IsBootstrapping == nil || !*h.Status.IsBootstrapping {
		return false
	}

	return true
}

// NumberOfReplicas permit to get the total of replicas
func (h *Elasticsearch) NumberOfReplicas() int32 {
	nbReplica := int32(0)
	for _, nodeGroup := range h.Spec.NodeGroups {
		nbReplica += nodeGroup.Replicas
	}

	return nbReplica
}
