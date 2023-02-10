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

package v1alpha1

import (
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FilebeatSpec defines the desired state of Filebeat
// +k8s:openapi-gen=true
type FilebeatSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// It will generate Elasticsearch output bas eon it
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef,omitempty"`

	// LogstashRef is the Logstash ref to connect on.
	// It will generate Logstash output base on it
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LogstashRef LogstashRef `json:"logstashRef,omitempty"`

	// Version is the Filebeat version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// Config is the Filebeat config
	// The key is the file stored on filebeat
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Module is the module specification
	// The key is the file stored on filebeat/modules.d
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Module map[string]string `json:"module,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`

	// Monitoring permit to monitor current cluster
	// Default, it not monitor cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Monitoring MonitoringSpec `json:"monitoring,omitempty"`

	// Ingresses permit to declare some ingresses
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingresses []Ingress `json:"ingresses,omitempty"`

	// Services permit to declare some services
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Services []Service `json:"services,omitempty"`
}

type LogstashRef struct {

	// ManagedLogstashRef is the managed Logstash instance by operator
	ManagedLogstashRef *LogstashManagedRef `json:"managed,omitempty"`

	// ExternalLogstahsRef is the external Logstash instance not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalLogstashRef *LogstashExternalRef `json:"external,omitempty"`

	// LogstashCaSecretRef is the secret that store your custom CA certificate to connect on Logstash via beat protocole.
	// It need to have the following keys: ca.crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LogstashCaSecretRef *corev1.LocalObjectReference `json:"logstashCASecretRef,omitempty"`
}

type LogstashManagedRef struct {

	// Name is the Logstash cluster deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`

	// Namespace is the namespace where Logstash is deployed by operator
	// No need to set if is deployed on the same namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// TargetService is the target service that expose the beat protocole
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetService string `json:"targetService,omitempty"`

	// Port is the port number to connect on service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Port int64 `json:"port,omitempty"`
}

type LogstashExternalRef struct {

	// Addresses is the list of Logstash addresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Addresses []string `json:"addresses,omitempty"`
}

type Ingress struct {

	// Name is the ingress name
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`

	// Spec is the ingress spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Spec networkingv1.IngressSpec `json:"spec,omitempty"`

	// Labels is the extra labels for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is the extra annotations for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// ContainerPortProtocol is the protocol to set when create service consumed by ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerPortProtocol corev1.Protocol `json:"containerProtocol,omitempty"`

	// ContainerPort is the port to set when create service consumed by ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerPort int64 `json:"containerPort,omitempty"`
}

type Service struct {

	// Name is the service name
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`

	// Spec is the service spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Spec corev1.ServiceSpec `json:"spec,omitempty"`

	// Labels is the extra labels for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is the extra annotations for ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

type MonitoringSpec struct {

	// Prometheus permit to monitor cluster with Prometheus and graphana (via exporter)
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
}

type PrometheusSpec struct {

	// Enabled permit to enable Prometheus monitoring
	// It will deploy exporter for filebeat and add podMonitor policy
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Url is the plugin URL where to download exporter
	// Default is use project https://github.com/pjhampton/kibana-prometheus-exporter
	// If version is set to latest, it use arbitrary: https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/8.6.0/kibanaPrometheusExporter-8.6.0.zip
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Url string `json:"url,omitempty"`
}

type DeploymentSpec struct {
	// Replicas is the number of replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas int32 `json:"replicas,omitempty"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *AntiAffinitySpec `json:"antiAffinity,omitempty"`

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
	PodTemplate *corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

	// PodDisruptionBudget is the pod disruption budget policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudgetSpec *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// Node permit to set extra option on Node process
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Jvm string `json:"jvm,omitempty"`

	// InitContainerResources permit to set resources on init containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	InitContainerResources *corev1.ResourceRequirements `json:"initContainerResources,omitempty"`

	// AdditionalVolumes permit to use additionnal volumes
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdditionalVolumes []VolumeSpec `json:"additionalVolumes,omitempty"`

	// Persistence is the spec to persist data
	// Default is emptyDir
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *PersistenceSpec `json:"persistence,omitempty"`

	// Ports is the list of container port to affect on filebeat container
	// It can be usefull to expose syslog input
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`
}

type PersistenceSpec struct {
	// VolumeClaim is the persistent volume claim spec use by statefullset
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	VolumeClaimSpec *corev1.PersistentVolumeClaimSpec `json:"volumeClaim,omitempty"`

	// Volume is the volume source to use instead volumeClaim
	// It usefull if you should to use hostPath
	// +optional
	Volume *corev1.VolumeSource `json:"volume,omitempty"`
}

type VolumeSpec struct {

	// Name is the volume name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`

	corev1.VolumeMount `json:",inline"`

	corev1.VolumeSource `json:",inline"`
}

type AntiAffinitySpec struct {

	// Type permit to set anti affinity as soft or hard
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type string `json:"type,omitempty"`

	// TopologyKey is the topology key to use
	// Default to topology.kubernetes.io/zone
	// +optional
	// +kubebuilder:default=topology.kubernetes.io/zone
	TopologyKey string `json:"topologyKey,omitempty"`
}

// FilebeatStatus defines the observed state of Filebeat
type FilebeatStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the current deployment phase
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Phase string `json:"phase"`

	// IsError is true if controller is stuck on Error
	// +operator-sdk:csv:customresourcedefinitions:type=status
	IsError bool `json:"isOnError"`

	// List of conditions
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Filebeat is the Schema for the filebeats API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Deployment status"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Filebeat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FilebeatSpec   `json:"spec,omitempty"`
	Status FilebeatStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FilebeatList contains a list of Filebeat
type FilebeatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filebeat `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Filebeat{}, &FilebeatList{})
}
