/*
Copyright 2024.

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
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/multiphase"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ElasticsearchAnnotationKey = "elasticsearch.discover.k8s.webcenter.fr"
)

// ElasticsearchSpec defines the desired state of Elasticsearch.
type ElasticsearchSpec struct {
	Discover `json:",inline"`

	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef,omitempty"`
}

// ElasticsearchStatus defines the observed state of Elasticsearch.
type ElasticsearchStatus struct {
	DiscoverStatus `json:",inline"`

	multiphase.DefaultMultiPhaseObjectStatus `json:",inline"`

	// ElasticsearchUserSecretRef is the secret that store all information to connect on Elasticsearch cluster
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ElasticsearchUserSecretRef *string `json:"elasticsearchUserSecretRef,omitempty"`

	// ElasticsearchCASecretRef is the secret that store CA certificate to connect on Elasticsearch cluster
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ElasticsearchCASecretRef *string `json:"elasticsearchCASecretRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:storageversion

// Elasticsearch is the Schema for the elasticsearches API.
// It will generate and maintain secret that store CA certificate for Elasticsearch.
// It will also maintain the URL to connect on Elasticsearch cluster.
// +operator-sdk:csv:customresourcedefinitions:resources={{Secret,v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="secretFile",type="string",JSONPath=".status.secretFileRef",description="secret ref that store certificate files"
// +kubebuilder:printcolumn:name="secretEnv",type="string",JSONPath=".status.secretEnvRef",description="secret ref that store env variables"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Elasticsearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticsearchSpec   `json:"spec,omitempty"`
	Status ElasticsearchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticsearchList contains a list of Elasticsearch.
type ElasticsearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Elasticsearch `json:"items"`
}
