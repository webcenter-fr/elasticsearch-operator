package shared

import corev1 "k8s.io/api/core/v1"

// Service permit to set service
type Service struct {
	// Name is the service name
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Spec is the service spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Spec corev1.ServiceSpec `json:"spec"`

	// Labels is the extra labels for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is the extra annotations for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}
