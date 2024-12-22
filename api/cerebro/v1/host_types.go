/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HostSpec defines the desired state of Host
// +k8s:openapi-gen=true
type HostSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CerebroRef is the Cerebro where to enroll Elasticsearch cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	CerebroRef HostCerebroRef `json:"cerebroRef"`

	// ElasticsearchRef is the Elasticsearch cluster to enroll on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef ElasticsearchRef `json:"elasticsearchRef"`
}

type HostCerebroRef struct {
	// Name is the cerebro name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Namespace is the cerebro namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type ElasticsearchRef struct {
	// ManagedElasticsearchRef is the managed Elasticsearch cluster by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedElasticsearchRef *corev1.LocalObjectReference `json:"managed,omitempty"`

	// ExternalElasticsearchRef is the external Elasticsearch cluster not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalElasticsearchRef *ElasticsearchExternalRef `json:"external,omitempty"`
}

type ElasticsearchExternalRef struct {
	// The cluster name to display on Cerabro
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name"`

	// Address is the public URL to access on Elasticsearch
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Address string `json:"address"`
}

// HostStatus defines the observed state of Host
type HostStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Host is the Schema for the hosts API
// +operator-sdk:csv:customresourcedefinitions:resources={{none,none}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Host struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostSpec   `json:"spec,omitempty"`
	Status HostStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HostList contains a list of Host
type HostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Host `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Host{}, &HostList{})
}
