package shared

import (
	"fmt"
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

	return nil
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
