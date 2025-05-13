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
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RoleMappingSpec defines the desired state of RoleMapping
// +k8s:openapi-gen=true
type RoleMappingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom role mapping name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Enabled permit to enable or disable the role mapping
	// Default to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Roles is the list of role to map
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Roles []string `json:"roles"`

	// Rules is the mapping rules
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Rules *apis.MapAny `json:"rules"`

	// Metadata is the meta data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *apis.MapAny `json:"metadata,omitempty"`
}

// RoleMappingStatus defines the observed state of RoleMapping
type RoleMappingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// RoleMapping is the Schema for the rolemappings API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type RoleMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleMappingSpec   `json:"spec,omitempty"`
	Status RoleMappingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleMappingList contains a list of RoleMapping
type RoleMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RoleMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RoleMapping{}, &RoleMappingList{})
}
