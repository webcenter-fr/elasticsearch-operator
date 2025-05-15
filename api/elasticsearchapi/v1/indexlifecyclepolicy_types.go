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
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IndexLifecyclePolicySpec defines the desired state of IndexLifecyclePolicy
// +k8s:openapi-gen=true
type IndexLifecyclePolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom index lifecycle policy name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Policy is the raw policy on JSON
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RawPolicy *string `json:"rawPolicy"`

	// Policy is the ILM policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Policy *IndexLifecyclePolicySpecPolicy `json:"policy,omitempty"`
}

type IndexLifecyclePolicySpecPolicy struct {
	// Metadata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Meta *apis.MapAny `json:"_meta,omitempty"`

	// ILM phases
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Phases IndexLifecyclePolicySpecPolicyPhases `json:"phases"`
}

type IndexLifecyclePolicySpecPolicyPhases struct {
	// Hot phase
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Hot *IndexLifecyclePolicySpecPolicyPhasesPhase `json:"hot,omitempty"`

	// Warm phase
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Warm *IndexLifecyclePolicySpecPolicyPhasesPhase `json:"warm,omitempty"`

	// Cold phase
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Cold *IndexLifecyclePolicySpecPolicyPhasesPhase `json:"cold,omitempty"`

	// Frozen phase
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Frozen *IndexLifecyclePolicySpecPolicyPhasesPhase `json:"frozen,omitempty"`

	// Delete phase
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Delete *IndexLifecyclePolicySpecPolicyPhasesPhase `json:"delete,omitempty"`
}

type IndexLifecyclePolicySpecPolicyPhasesPhase struct {
	// The min age to exec action
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MinAge *string `json:"min_age,omitempty"`

	// The ILM action
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Actions apis.MapAny `json:"actions"`
}

// IndexLifecyclePolicyStatus defines the observed state of IndexLifecyclePolicy
type IndexLifecyclePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// IndexLifecyclePolicy is the Schema for the indexlifecyclepolicies API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type IndexLifecyclePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IndexLifecyclePolicySpec   `json:"spec,omitempty"`
	Status IndexLifecyclePolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IndexLifecyclePolicyList contains a list of IndexLifecyclePolicy
type IndexLifecyclePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IndexLifecyclePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IndexLifecyclePolicy{}, &IndexLifecyclePolicyList{})
}
