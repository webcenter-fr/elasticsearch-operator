package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Elasticsearch) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// IsSelfManagedSecretForTlsApi return true if the operator manage the certificates for Api layout
// It return false if secret is provided
func (h *Elasticsearch) IsSelfManagedSecretForTlsApi() bool {
	return h.Spec.Tls.CertificateSecretRef == nil
}

// IsTlsApiEnabled return true if TLS is enabled on API endpoint
func (h *Elasticsearch) IsTlsApiEnabled() bool {
	if h.Spec.Tls.Enabled != nil && !*h.Spec.Tls.Enabled {
		return false
	}
	return true
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

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Elasticsearch) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *Elasticsearch) IsMetricbeatMonitoring() bool {

	// compute the numbers of replica
	nbReplica := int32(0)
	for _, nodeGroup := range h.Spec.NodeGroups {
		nbReplica += nodeGroup.Replicas
	}

	if h.Spec.Monitoring.Metricbeat != nil && h.Spec.Monitoring.Metricbeat.Enabled && nbReplica > 0 {
		return true
	}

	return false
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
