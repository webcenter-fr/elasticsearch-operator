package shared

import corev1 "k8s.io/api/core/v1"

// TlsSpec permit to set TLS
type TlsSpec struct {
	// Enabled permit to enabled TLS
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// SelfSignedCertificate permit to set self signed certificate settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SelfSignedCertificate *TlsSelfSignedCertificateSpec `json:"selfSignedCertificate,omitempty"`

	// CertificateSecretRef is the secret that store your custom certificates.
	// It need to have the following keys: tls.key, tls.crt and optionally ca.crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	CertificateSecretRef *corev1.LocalObjectReference `json:"certificateSecretRef,omitempty"`

	// ValidityDays is the number of days that certificates are valid
	// Default to 365
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=365
	ValidityDays *int `json:"validityDays,omitempty"`

	// RenewalDays is the number of days before certificate expire to become effective renewal
	// Default to 30
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=30
	RenewalDays *int `json:"renewalDays,omitempty"`

	// KeySize is the key size when generate privates keys
	// Default to 2048
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=2048
	KeySize *int `json:"keySize,omitempty"`
}

// TlsSelfSignedCertificateSpec permit to set the the self signed certificate
type TlsSelfSignedCertificateSpec struct {
	// AltIps permit to set subject alt names of type ip when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltIps []string `json:"altIPs,omitempty"`

	// AltNames permit to set subject alt names of type dns when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltNames []string `json:"altNames,omitempty"`
}

// IsSelfManagedSecretForTls return true if the operator manage the certificates for TLS
// It return false if secret is provided
func (h TlsSpec) IsSelfManagedSecretForTls() bool {
	return h.CertificateSecretRef == nil
}

// IsTlsEnabled return true if TLS is enabled to access on Kibana
func (h TlsSpec) IsTlsEnabled() bool {
	if h.Enabled != nil && !*h.Enabled {
		return false
	}
	return true
}
