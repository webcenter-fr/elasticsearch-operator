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

// RoleSpec defines the desired state of Role
// +k8s:openapi-gen=true
type RoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom role name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Cluster is a list of cluster privileges
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Cluster []string `json:"cluster,omitempty"`

	// Indices is the list of indices permissions
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Indices []RoleSpecIndicesPermissions `json:"indices,omitempty"`

	// Applications is the list of application privilege
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Applications []RoleSpecApplicationPrivileges `json:"applications,omitempty"`

	// RunAs is the list of users that the owners of this role can impersonate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RunAs []string `json:"runAs,omitempty"`

	// Global  defining global privileges
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Global *apis.MapAny `json:"global,omitempty"`

	// Metadata is optional meta-data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *apis.MapAny `json:"metadata,omitempty"`

	// TransientMetadata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	TransientMetadata *apis.MapAny `json:"transientMetadata,omitempty"`
}

// ElasticsearchRoleSpecApplicationPrivileges is the application privileges object
type RoleSpecApplicationPrivileges struct {
	// Application
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Application string `json:"application"`

	// Privileges
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Privileges []string `json:"privileges,omitempty"`

	// Resources
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources []string `json:"resources,omitempty"`
}

// RoleSpecIndicesPermissions is the indices permission object
type RoleSpecIndicesPermissions struct {
	// Names
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Names []string `json:"names"`

	// Privileges
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Privileges []string `json:"privileges"`

	// FieldSecurity
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	FieldSecurity *apis.MapAny `json:"fieldSecurity,omitempty"`

	// Query
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Query string `json:"query,omitempty"`

	// Allow to manage restricted index
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AllowRestrictedIndices bool `json:"allowRestrictedIndices,omitempty"`
}

// RoleStatus defines the observed state of Role
type RoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Role is the Schema for the roles API
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleSpec   `json:"spec,omitempty"`
	Status RoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleList contains a list of Role
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
