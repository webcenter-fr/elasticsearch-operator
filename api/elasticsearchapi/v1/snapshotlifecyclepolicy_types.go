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

// SnapshotLifecyclePolicySpec defines the desired state of SnapshotLifecyclePolicy
// +k8s:openapi-gen=true
type SnapshotLifecyclePolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// SnapshotLifecyclePolicyName is the custom snapshot lifecycle policy name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SnapshotLifecyclePolicyName string `json:"snapshotLifecyclePolicyName,omitempty"`

	// Schedule is schedule policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Schedule string `json:"schedule"`

	// Name is the template name to generte final name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Repository is the target repository to store backup
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Repository string `json:"repository"`

	// Config is the config backup
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Config SLMConfig `json:"config"`

	// Retention is the retention policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Retention *SLMRetention `json:"retention,omitempty"`
}

// SLMConfig is the config sub section
type SLMConfig struct {
	// ExpendWildcards
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExpendWildcards string `json:"expandWildcards,omitempty"`

	// IgnoreUnavailable
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IgnoreUnavailable bool `json:"ignoreUnavailable,omitempty"`

	// IncludeGlobalState
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IncludeGlobalState bool `json:"includeGlobalState,omitempty"`

	// Indices
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Indices []string `json:"indices,omitempty"`

	// FeatureStates
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	FeatureStates []string `json:"featureStates,omitempty"`

	// Metadata
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *apis.MapAny `json:"metadata,omitempty"`

	// Partial
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Partial bool `json:"partial,omitempty"`
}

type SLMRetention struct {
	// ExpireAfter
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExpireAfter string `json:"expireAfter,omitempty"`

	// MaxCount
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MaxCount int64 `json:"maxCount,omitempty"`

	// MinCount
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MinCount int64 `json:"minCount,omitempty"`
}

// SnapshotLifecyclePolicyStatus defines the observed state of SnapshotLifecyclePolicy
type SnapshotLifecyclePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	remote.DefaultRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// SnapshotLifecyclePolicy is the Schema for the snapshotlifecyclepolicies API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type SnapshotLifecyclePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotLifecyclePolicySpec   `json:"spec,omitempty"`
	Status SnapshotLifecyclePolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SnapshotLifecyclePolicyList contains a list of SnapshotLifecyclePolicy
type SnapshotLifecyclePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnapshotLifecyclePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SnapshotLifecyclePolicy{}, &SnapshotLifecyclePolicyList{})
}
