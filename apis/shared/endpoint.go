package shared

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// EndpointSpec permit to set endpoint
type EndpointSpec struct {
	// Ingress permit to set ingress settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingress *EndpointIngressSpec `json:"ingress,omitempty"`

	// Load balancer permit to set load balancer settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LoadBalancer *EndpointLoadBalancerSpec `json:"loadBalancer,omitempty"`
}

// EndpointLoadBalancerSpec permit to set Load balancer
type EndpointLoadBalancerSpec struct {
	// Enabled permit to enabled / disabled load balancer
	// Cloud provider need to support it
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

// EndpointIngressSpec permit to set endpoint
type EndpointIngressSpec struct {

	// Enabled permit to enabled / disabled ingress
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Host is the hostname to access on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Host string `json:"host"`

	// SecretRef is the secret ref that store certificates
	// If secret not exist, it will use the default ingress certificate
	// If secret is nil, it will disable TLS on ingress spec.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// Labels to set in ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to set in ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// IngressSpec it merge with expected ingress spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IngressSpec *networkingv1.IngressSpec `json:"ingressSpec,omitempty"`
}
