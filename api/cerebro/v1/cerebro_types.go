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

const (
	CerebroAnnotationKey = "cerebro.k8s.webcenter.fr"
)

// CerebroSpec defines the desired state of Cerebro
// +k8s:openapi-gen=true
type CerebroSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// Version is the Cerebro version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// Endpoint permit to set endpoints to access on Cerebro from external kubernetes
	// You can set ingress and / or load balancer
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Endpoint shared.EndpointSpec `json:"endpoint,omitempty"`

	// Config is the Cerebro config
	// The key is the file stored on config folders and the data is the contend
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config *string `json:"config,omitempty"`

	// ExtraConfigs is extra config files store on config directory
	// The key is the file name and the value is the contend file
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExtraConfigs map[string]string `json:"extraConfigs,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Deployment CerebroDeploymentSpec `json:"deployment,omitempty"`
}

type CerebroDeploymentSpec struct {
	shared.Deployment `json:",inline"`

	// Node permit to set extra option on Node process
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Node string `json:"node,omitempty"`
}

// CerebroStatus defines the observed state of Cerebro
type CerebroStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`

	// Url is the Cerebro endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Url string `json:"url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Cerebro is the Schema for the cerebroes API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Deployment,apps/v1}}
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Cerebro struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CerebroSpec   `json:"spec,omitempty"`
	Status CerebroStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CerebroList contains a list of Cerebro
type CerebroList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cerebro `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cerebro{}, &CerebroList{})
}
