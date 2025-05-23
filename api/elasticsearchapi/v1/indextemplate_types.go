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

// IndexTemplateSpec defines the desired state of IndexTemplate
// +k8s:openapi-gen=true
type IndexTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom index template name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// IndexPatterns is the list of index to apply this template
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IndexPatterns []string `json:"indexPatterns,omitempty"`

	// ComposedOf is the list of component templates
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ComposedOf []string `json:"composedOf,omitempty"`

	// Priority is the priority to apply this template
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Priority int `json:"priority,omitempty"`

	// The version
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Version int `json:"version,omitempty"`

	// Template is the template specification
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Template *IndexTemplateData `json:"template,omitempty"`

	// Meta is extended info as JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Meta *apis.MapAny `json:"meta,omitempty"`

	// AllowAutoCreate permit to allow auto create index
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AllowAutoCreate bool `json:"allowAutoCreate,omitempty"`

	// RawTemplate is the raw template
	// You can use it instead to set indexPatterns, composedOf, priority, template etc.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RawTemplate *string `json:"rawTemplate,omitempty"`
}

// IndexTemplateData is the template specification
type IndexTemplateData struct {
	// Settings is the template setting as JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Settings *apis.MapAny `json:"settings,omitempty"`

	// Mappings is the template mapping as JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Mappings *apis.MapAny `json:"mappings,omitempty"`

	// Aliases is the template alias as JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Aliases *apis.MapAny `json:"aliases,omitempty"`
}

// IndexTemplateStatus defines the observed state of IndexTemplate
type IndexTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// IndexTemplate is the Schema for the indextemplates API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type IndexTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IndexTemplateSpec   `json:"spec,omitempty"`
	Status IndexTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IndexTemplateList contains a list of IndexTemplate
type IndexTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IndexTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IndexTemplate{}, &IndexTemplateList{})
}
