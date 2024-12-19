package shared

import (
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
)

// Route permit to set route
type Route struct {
	// Name is the route name
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Spec is the route spec
	// @clean
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Spec routev1.RouteSpec `json:"spec"`

	// Labels is the extra labels for route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is the extra annotations for route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// ContainerPortProtocol is the protocol to set when create service consumed by route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerPortProtocol corev1.Protocol `json:"containerProtocol"`

	// ContainerPort is the port to set when create service consumed by route
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerPort int64 `json:"containerPort"`
}
