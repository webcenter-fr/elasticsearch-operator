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
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogstashPipelineSpec defines the desired state of LogstashPipeline
// +k8s:openapi-gen=true
type LogstashPipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// KibanaRef is the Kibana ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KibanaRef shared.KibanaRef `json:"kibanaRef"`

	// Name is the Logstash pipeline ID
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Description is the pipeline description
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Description string `json:"description,omitempty"`

	// Pipeline is the pipeline definition
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Pipeline string `json:"pipeline"`

	// Settings is the pipeline settings
	// It's a JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Settings string `json:"settings,omitempty"`
}

// LogstashPipelineStatus defines the observed state of LogstashPipeline
type LogstashPipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// LogstashPipeline is the Schema for the logstashpipelines API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type LogstashPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogstashPipelineSpec   `json:"spec,omitempty"`
	Status LogstashPipelineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LogstashPipelineList contains a list of LogstashPipeline
type LogstashPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogstashPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogstashPipeline{}, &LogstashPipelineList{})
}
