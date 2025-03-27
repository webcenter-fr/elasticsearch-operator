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

// RoleSpec defines the desired state of Role
// +k8s:openapi-gen=true
type RoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// KibanaRef is the Kibana ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KibanaRef shared.KibanaRef `json:"kibanaRef"`

	// Name is the role name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Elasticsearch is the Elasticsearch right
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Elasticsearch *KibanaRoleElasticsearch `json:"elasticsearch,omitempty"`

	// Kibana is the Kibana right
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Kibana []KibanaRoleKibana `json:"kibana,omitempty"`

	// TransientMedata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TransientMedata *KibanaRoleTransientMetadata `json:"transientMetadata,omitempty"`

	// Metadata is optional meta-data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *apis.MapAny `json:"metadata,omitempty"`
}

type KibanaRoleTransientMetadata struct {
	// Enabled permit to enable transient metadata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Enabled bool `json:"enabled,omitempty"`
}

type KibanaRoleElasticsearch struct {
	// Indices is the indice privileges
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Indices []KibanaRoleElasticsearchIndice `json:"indices,omitempty"`

	// Cluster is the cluster privilege
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Cluster []string `json:"cluster,omitempty"`

	// RunAs is the privilege like users
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RunAs []string `json:"runAs,omitempty"`
}

type KibanaRoleKibana struct {
	// Base is the base privilege
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Base []string `json:"base,omitempty"`

	// Feature ontains privileges for specific features
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Feature map[string][]string `json:"feature,omitempty"`

	// Spaces is the list of space o apply the privileges to
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Spaces []string `json:"spaces,omitempty"`
}

type KibanaRoleElasticsearchIndice struct {
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
}

// RoleStatus defines the observed state of Role
type RoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Role is the Schema for the roles API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
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
