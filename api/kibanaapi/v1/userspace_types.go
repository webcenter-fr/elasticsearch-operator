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
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UserSpaceSpec defines the desired state of UserSpace
// +k8s:openapi-gen=true
type UserSpaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// KibanaRef is the Kibana ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KibanaRef shared.KibanaRef `json:"kibanaRef"`

	// ID is the user space ID
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the user space name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Description is the user space description
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Description string `json:"description,omitempty"`

	// DisabledFeatures is the list of feature disabled on current user space
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	DisabledFeatures []string `json:"disabledFeatures,omitempty"`

	// Initials is the user space initials
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Initials string `json:"initials,omitempty"`

	// Color is the user space color
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Color string `json:"color,omitempty"`

	// CopyObjects permit to copy objects into current user space
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KibanaUserSpaceCopies []KibanaUserSpaceCopy `json:"userSpaceCopies,omitempty"`
}

type KibanaUserSpaceCopy struct {
	// OriginUserSpace is the user space name from copy objects
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	OriginUserSpace string `json:"originUserSpace"`

	// IncludeReferences is set to true to copy all references
	// Default to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IncludeReferences *bool `json:"includeReferences,omitempty"`

	// Overwrite is set to true to overwrite existing objects
	// Default to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Overwrite *bool `json:"overwrite,omitempty"`

	// CreateNewCopies is set to true to create new copy of objects
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	CreateNewCopies *bool `json:"createNewCopies,omitempty"`

	// ForceUpdateWhenReconcile is set to true to force to sync objects each time the operator reconcile
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ForceUpdateWhenReconcile *bool `json:"forceUpdate,omitempty"`

	// KibanaObjects is the list of object to copy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KibanaObjects []KibanaSpaceObjectParameter `json:"objects"`
}

type KibanaSpaceObjectParameter struct {
	// Tpye is the object type to copy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type string `json:"type"`

	// ID is the object to copy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ID string `json:"id"`
}

// UserSpaceStatus defines the observed state of UserSpace
type UserSpaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// UserSpace is the Schema for the userspaces API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type UserSpace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpaceSpec   `json:"spec,omitempty"`
	Status UserSpaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserSpaceList contains a list of UserSpace
type UserSpaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserSpace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UserSpace{}, &UserSpaceList{})
}
