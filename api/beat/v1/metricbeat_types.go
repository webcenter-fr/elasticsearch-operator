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
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/multiphase"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// +k8s:openapi-gen=true

const (
	MetricbeatAnnotationKey = "metricbeat.k8s.webcenter.fr"
)

// MetricbeatSpec defines the desired state of Metricbeat
type MetricbeatSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// It will generate Elasticsearch output bas on it
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Version is the Metricbeat version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// Config is the Metricbeat config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config *apis.MapAny `json:"config,omitempty"`

	// ExtraConfigs is extra config files store on config directory
	// The key is the file name and the value is the contend file
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExtraConfigs map[string]string `json:"extraConfigs,omitempty"`

	// Module is the module specification
	// The key is the file stored on metricbeat/modules.d
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Modules *apis.MapAny `json:"modules,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Deployment MetricbeatDeploymentSpec `json:"deployment"`
}

type MetricbeatDeploymentSpec struct {
	shared.Deployment `json:",inline"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *shared.DeploymentAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// PodDisruptionBudget is the pod disruption budget policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudgetSpec *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// InitContainerResources permit to set resources on init containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	InitContainerResources *corev1.ResourceRequirements `json:"initContainerResources,omitempty"`

	// AdditionalVolumes permit to use additionnal volumes
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdditionalVolumes []shared.DeploymentVolumeSpec `json:"additionalVolumes,omitempty"`

	// Persistence is the spec to persist data
	// Default is emptyDir
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *shared.DeploymentPersistenceSpec `json:"persistence,omitempty"`
}

type MetricbeatPersistenceSpec struct {
	// VolumeClaim is the persistent volume claim spec use by statefullset
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	VolumeClaimSpec *corev1.PersistentVolumeClaimSpec `json:"volumeClaim,omitempty"`

	// Volume is the volume source to use instead volumeClaim
	// It usefull if you should to use hostPath
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Volume *corev1.VolumeSource `json:"volume,omitempty"`
}

// MetricbeatStatus defines the observed state of Metricbeat
type MetricbeatStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	multiphase.DefaultMultiPhaseObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Metricbeat is the Schema for the metricbeats API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{StatefulSet,apps/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1},{ServiceAccount,v1},{RoleBinding,rbac.authorization.k8s.io/v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Metricbeat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricbeatSpec   `json:"spec,omitempty"`
	Status MetricbeatStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MetricbeatList contains a list of Metricbeat
type MetricbeatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Metricbeat `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Metricbeat{}, &MetricbeatList{})
}
