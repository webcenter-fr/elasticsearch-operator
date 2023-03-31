package shared

import (
	corev1 "k8s.io/api/core/v1"
)

type KibanaRef struct {

	// ManagedKibanaRef is the managed Kibana by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedKibanaRef *KibanaManagedRef `json:"managed,omitempty"`

	// ExternalKibanaRef is the external Kibana not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalKibanaRef *KibanaExternalRef `json:"external,omitempty"`

	// KibanaCaSecretRef is the secret that store your custom CA certificate to connect on Kibana API.
	// It need to have the following keys: ca.crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KibanaCaSecretRef *corev1.LocalObjectReference `json:"kibanaCASecretRef,omitempty"`

	// KibanaCredentialSecretRef is the secret that contain credential to acess on Kibana API.
	// It need to contain the keys `username` and `password`.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KibanaCredentialSecretRef *corev1.LocalObjectReference `json:"credentialSecretRef,omitempty"`
}

type KibanaManagedRef struct {

	// Name is the Kibana deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Namespace is the namespace where Kibana is deployed by operator
	// No need to set if Kibana is deployed on the same namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type KibanaExternalRef struct {

	// Address is the Kibana address
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Address string `json:"address"`
}

// IsManaged permit to know if Kibana is managed by operator
func (h KibanaRef) IsManaged() bool {
	return h.ManagedKibanaRef != nil && h.ManagedKibanaRef.Name != ""
}

// IsExternal permit to know if Kibana is external (not managed by operator)
func (h KibanaRef) IsExternal() bool {
	return h.ExternalKibanaRef != nil && h.ExternalKibanaRef.Address != ""
}
