package shared

import (
	corev1 "k8s.io/api/core/v1"
)

type ElasticsearchRef struct {

	// ManagedElasticsearchRef is the managed Elasticsearch cluster by operator
	ManagedElasticsearchRef *ElasticsearchManagedRef `json:"managed,omitempty"`

	// ExternalElasticsearchRef is the external Elasticsearch cluster not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalElasticsearchRef *ElasticsearchExternalRef `json:"external,omitempty"`
}

type ElasticsearchManagedRef struct {

	// Name is the Elasticsearch cluster deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`

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
	Addresses []string `json:"addresses,omitempty"`

	// SecretName is the secret that contain the setting to connect on Elasticsearch that is not managed by ECK.
	// It need to contain only one entry. The user is the key, and the password is the data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// IsManaged permit to know if Elasticsearch is managed by operator
func (h ElasticsearchRef) IsManaged() bool {
	return h.ManagedElasticsearchRef != nil && h.ManagedElasticsearchRef.Name != ""
}

// IsExternal permit to know if Elasticsearch is external (not managed by operator)
func (h ElasticsearchRef) IsExternal() bool {
	return h.ExternalElasticsearchRef != nil && len(h.ExternalElasticsearchRef.Addresses) > 0
}
