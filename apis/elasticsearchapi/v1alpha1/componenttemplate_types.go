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

package v1alpha1

import (
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ComponentTemplateSpec defines the desired state of ComponentTemplate
// +k8s:openapi-gen=true
type ComponentTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef,omitempty"`

	// Name is the custom component template name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Settings is the component setting
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Settings string `json:"settings,omitempty"`

	// Mappings is the component mapping
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Mappings string `json:"mappings,omitempty"`

	// Aliases is the component aliases
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Aliases string `json:"aliases,omitempty"`
}

// ComponentTemplateStatus defines the observed state of ComponentTemplate
type ComponentTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// List of conditions
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions"`

	// Health
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Health bool `json:"health"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ComponentTemplate is the Schema for the componenttemplates API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Health",type="boolean",JSONPath=".status.health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ComponentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentTemplateSpec   `json:"spec,omitempty"`
	Status ComponentTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ComponentTemplateList contains a list of ComponentTemplate
type ComponentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ComponentTemplate{}, &ComponentTemplateList{})
}
