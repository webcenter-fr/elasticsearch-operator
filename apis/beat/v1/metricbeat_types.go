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
	// The key is the file stored on metricbeat
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Module is the module specification
	// The key is the file stored on metricbeat/modules.d
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Module map[string]string `json:"module,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Deployment MetricbeatDeploymentSpec `json:"deployment"`
}

type MetricbeatDeploymentSpec struct {
	// Replicas is the number of replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas int32 `json:"replicas"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *MetricbeatAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// Resources permit to set ressources on container
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations permit to set toleration on pod
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// NodeSelector permit to set node selector on pod
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Labels permit to set labels on containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations permit to set annotation on deployment
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Env permit to set some environment variable on containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom permit to set some environment variable from config map or secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// PodSpec is merged with expected pod
	// It usefull to add some extra properties on pod spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

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
	AdditionalVolumes []MetricbeatVolumeSpec `json:"additionalVolumes,omitempty"`

	// Persistence is the spec to persist data
	// Default is emptyDir
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *MetricbeatPersistenceSpec `json:"persistence,omitempty"`
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

type MetricbeatVolumeSpec struct {

	// Name is the volume name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	corev1.VolumeMount `json:",inline"`

	corev1.VolumeSource `json:",inline"`
}

type MetricbeatAntiAffinitySpec struct {

	// Type permit to set anti affinity as soft or hard
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type string `json:"type"`

	// TopologyKey is the topology key to use
	// Default to topology.kubernetes.io/zone
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=topology.kubernetes.io/zone
	TopologyKey string `json:"topologyKey,omitempty"`
}

// MetricbeatStatus defines the observed state of Metricbeat
type MetricbeatStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Metricbeat is the Schema for the metricbeats API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{StatefulSet,apps/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1}}
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
