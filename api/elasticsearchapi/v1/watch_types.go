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

// WatchSpec defines the desired state of Watch
// +k8s:openapi-gen=true
type WatchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom watch name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Trigger
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Trigger *apis.MapAny `json:"trigger"`

	// Input
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Input *apis.MapAny `json:"input"`

	// Conditiong
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Condition *apis.MapAny `json:"condition"`

	// Transform
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Transform *apis.MapAny `json:"transform,omitempty"`

	// ThrottlePeriod
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ThrottlePeriod string `json:"throttle_period,omitempty"`

	// ThrottlePeriodInMillis
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ThrottlePeriodInMillis int64 `json:"throttle_period_in_millis,omitempty"`

	// Actions
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	Actions *apis.MapAny `json:"actions"`

	// Metadata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *apis.MapAny `json:"metadata,omitempty"`
}

// WatchStatus defines the observed state of Watch
type WatchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Watch is the Schema for the watches API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Watch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WatchSpec   `json:"spec,omitempty"`
	Status WatchStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WatchList contains a list of Watch
type WatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Watch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Watch{}, &WatchList{})
}
