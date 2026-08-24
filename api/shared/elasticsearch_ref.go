package shared

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type ElasticsearchRef struct {
	// ManagedElasticsearchRef is the managed Elasticsearch cluster by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedElasticsearchRef *ElasticsearchManagedRef `json:"managed,omitempty"`

	// ExternalElasticsearchRef is the external Elasticsearch cluster not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalElasticsearchRef *ElasticsearchExternalRef `json:"external,omitempty"`

	// ElasticsearchCaSecretRef is the secret that store your custom CA certificate to connect on Elasticsearch API.
	// It need to have the following keys: ca.crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ElasticsearchCaSecretRef *corev1.LocalObjectReference `json:"elasticsearchCASecretRef,omitempty"`

	// SecretName is the secret that contain the setting to connect on Elasticsearch. It can be auto computed for managed Elasticsearch.
	// It need to contain the keys `username` and `password`.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

type ElasticsearchManagedRef struct {
	// Name is the Elasticsearch cluster deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Namespace is the namespace where Elasticsearch is deployed by operator
	// No need to set if Kibana is deployed on the same namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// TargetNodeGroup is the target Elasticsearch node group to use as service to connect on Elasticsearch
	// Default, it use the global service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroup string `json:"targetNodeGroup,omitempty"`
}

type ElasticsearchExternalRef struct {
	// Addresses is the list of Elasticsearch addresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Addresses []string `json:"addresses"`
}

// IsManaged permit to know if Elasticsearch is managed by operator
func (h ElasticsearchRef) IsManaged() bool {
	return h.ManagedElasticsearchRef != nil && h.ManagedElasticsearchRef.Name != ""
}

// IsExternal permit to know if Elasticsearch is external (not managed by operator)
func (h ElasticsearchRef) IsExternal() bool {
	return h.ExternalElasticsearchRef != nil && len(h.ExternalElasticsearchRef.Addresses) > 0
}

// ValidateField permit to validate field from webhook
func (h ElasticsearchRef) ValidateField() *field.Error {
	// Check we provide Opensearch cluster
	if !h.IsExternal() && !h.IsManaged() {
		return field.Required(field.NewPath("spec").Child("elasticsearchRef"), "You need to provide managed or external Elasticsearch cluster")
	}

	// SSRF hardening: every external address must be a well-formed http(s) URL
	// with a non-empty host and must not target the cloud metadata endpoint.
	if h.IsExternal() {
		for i, address := range h.ExternalElasticsearchRef.Addresses {
			p := field.NewPath("spec").Child("elasticsearchRef").Child("external").Child("addresses").Index(i)
			if err := validateExternalElasticsearchAddress(address); err != nil {
				return field.Invalid(p, address, err.Error())
			}
		}
	}

	return nil
}

// cloudMetadataIPv4 is the link-local IPv4 address used by cloud metadata
// services (AWS, GCP, Azure, Alibaba, ...). It must never be dialable through
// an external Elasticsearch ref.
var cloudMetadataIPv4 = net.IPv4(169, 254, 169, 254)

// validateExternalElasticsearchAddress validates a single external Elasticsearch
// address. It does not block private/loopback ranges (on-prem Elasticsearch must
// keep working); it only rejects malformed URLs and the cloud metadata endpoint.
func validateExternalElasticsearchAddress(address string) error {
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("address must be a valid URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("address must use the http or https scheme")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("address must have a non-empty host")
	}
	if targetsCloudMetadata(host) {
		return fmt.Errorf("address must not target the cloud metadata endpoint (169.254.169.254)")
	}
	return nil
}

// targetsCloudMetadata reports whether the URL host is the link-local metadata
// address in any of its textual forms: dotted-quad IPv4, or an IPv6 literal that
// embeds it (e.g. the IPv4-mapped ::ffff:169.254.169.254, or the hex form
// ::ffff:a9fe:a9fe). net.ParseIP normalizes all of these to the same 4-byte IPv4
// address via To4, so an attacker cannot bypass the block with an IPv6-mapped or
// hexadecimal spelling of the metadata IP.
func targetsCloudMetadata(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.Equal(cloudMetadataIPv4)
	}
	return false
}

// GetTargetCluster permit to get the target cluster
func (h ElasticsearchRef) GetTargetCluster(currentNamespace string) string {
	if currentNamespace == "" {
		panic("You must provide currentNamespace")
	}
	if h.IsManaged() {
		namespace := currentNamespace
		if h.ManagedElasticsearchRef.Namespace != "" {
			namespace = h.ManagedElasticsearchRef.Namespace
		}
		return fmt.Sprintf("%s/%s", namespace, h.ManagedElasticsearchRef.Name)
	} else if h.IsExternal() {
		return strings.Join(h.ExternalElasticsearchRef.Addresses, ",")
	}

	return ""
}
