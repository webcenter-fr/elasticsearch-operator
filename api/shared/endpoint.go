package shared

import (
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// EndpointSpec permit to set endpoint
type EndpointSpec struct {
	// Ingress permit to set ingress settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingress *EndpointIngressSpec `json:"ingress,omitempty"`

	// Route permit to set route settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Route *EndpointRouteSpec `json:"route,omitempty"`

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

	// Set to true to enable TLS on Ingress
	// Default to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	TlsEnabled *bool `json:"tlsEnabled,omitempty"`

	// SecretRef is the secret ref that store certificates
	// If secret not exist, it will use the default ingress certificate
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

// EndpointIngressSpec permit to set route endpoint
type EndpointRouteSpec struct {
	// Enabled permit to enabled / disabled route
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Host is the hostname to access on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Host string `json:"host"`

	// Set to true to enable TLS on route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TlsEnabled *bool `json:"tlsEnabled,omitempty"`

	// SecretRef is the secret ref that store certificates
	// If secret not exist, it will use the default route certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// Labels to set in route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to set in route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// RouteSpec it merge with expected route spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RouteSpec *routev1.RouteSpec `json:"routeSpec,omitempty"`
}

// IsIngressEnabled return true if ingress is enabled
func (h EndpointSpec) IsIngressEnabled() bool {
	if h.Ingress != nil && h.Ingress.Enabled {
		return true
	}

	return false
}

// IsRouteEnabled return true if route is enabled
func (h EndpointSpec) IsRouteEnabled() bool {
	if h.Route != nil && h.Route.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h EndpointSpec) IsLoadBalancerEnabled() bool {
	if h.LoadBalancer != nil && h.LoadBalancer.Enabled {
		return true
	}

	return false
}

// IsTlsEnabled return true if TLS is enabled
// If TlsEnabled is not set, it will return true
// If TlsEnabled is set to false, it will return false
// If TlsEnabled is set to true, it will return true
func (h EndpointIngressSpec) IsTlsEnabled() bool {
	if h.TlsEnabled != nil {
		return *h.TlsEnabled
	}

	return true
}

// IsTlsEnabled return true if TLS is enabled
// If TlsEnabled is not set, it will return true
// If TlsEnabled is set to false, it will return false
// If TlsEnabled is set to true, it will return true
func (h EndpointRouteSpec) IsTlsEnabled() bool {
	if h.TlsEnabled != nil {
		return *h.TlsEnabled
	}

	return true
}
