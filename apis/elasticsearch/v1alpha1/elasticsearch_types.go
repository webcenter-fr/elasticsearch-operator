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

const (
	ElasticsearchAnnotationKey = "elasticsearch.k8s.webcenter.fr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticsearchSpec defines the desired state of Elasticsearch
// +k8s:openapi-gen=true
type ElasticsearchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// Version is the Elasticsearch version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// ClusterName is the Elasticsearch cluster name
	// Default is use the custom ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// SetVMMaxMapCount permit to set the right value for VMMaxMapCount on node
	// It need to run pod as root with privileged option
	// Default is true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	SetVMMaxMapCount *bool `json:"setVMMaxMapCount,omitempty"`

	// PluginsList is the list of additionnal plugin to install on each Elasticsearch node
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PluginsList []string `json:"pluginsList,omitempty"`

	// GlobalNodeGroup permit to set some default parameters for each node groups
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	GlobalNodeGroup ElasticsearchGlobalNodeGroupSpec `json:"globalNodeGroup,omitempty"`

	// NodeGroups permit to groups node per use case
	// For exemple master, data and ingest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	NodeGroups []ElasticsearchNodeGroupSpec `json:"nodeGroups,omitempty"`

	// Endpoint permit to set endpoints to access on Elasticsearch from external kubernetes
	// You can set ingress and / or load balancer
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Endpoint ElasticsearchEndpointSpec `json:"endpoint,omitempty"`

	// Tls permit to set the TLS setting for API access
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Tls ElasticsearchTlsSpec `json:"tls,omitempty"`

	// LicenseSecretRef permit to set secret that contain Elasticsearch license on key `license`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LicenseSecretRef *corev1.LocalObjectReference `json:"licenseSecretRef,omitempty"`

	// Monitoring permit to monitor current cluster
	// Default, it not monitor cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Monitoring ElasticsearchMonitoringSpec `json:"monitoring,omitempty"`
}

type ElasticsearchMonitoringSpec struct {

	// Prometheus permit to monitor cluster with Prometheus and graphana (via exporter)
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Prometheus *ElasticsearchPrometheusSpec `json:"prometheus,omitempty"`

	// Metricbeat permit to monitor cluster with metricbeat and to dedicated monitoring cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Metricbeat *shared.MetricbeatMonitoringSpec `json:"metricbeat,omitempty"`
}

type ElasticsearchPrometheusSpec struct {
	shared.ImageSpec `json:",inline"`

	// Enabled permit to enable Prometheus monitoring
	// It will deploy exporter for Elasticsearch and add podMonitor policy
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Version is the exporter version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`
}

type ElasticsearchEndpointSpec struct {
	// Ingress permit to set ingress settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingress *ElasticsearchIngressSpec `json:"ingress,omitempty"`

	// Load balancer permit to set load balancer settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LoadBalancer *ElasticsearchLoadBalancerSpec `json:"loadBalancer,omitempty"`
}

type ElasticsearchLoadBalancerSpec struct {
	// Enabled permit to enabled / disabled load balancer
	// Cloud provider need to support it
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// TargetNodeGroupName permit to define if specific node group is responsible to receive external access, like ingest nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroupName string `json:"targetNodeGroupName,omitempty"`
}

type ElasticsearchTlsSpec struct {

	// Enabled permit to enabled TLS on API
	// Default true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// SelfSignedCertificate permit to set self signed certificate settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SelfSignedCertificate *ElasticsearchSelfSignedCertificateSpec `json:"selfSignedCertificate,omitempty"`

	// CertificateSecretRef is the secret that store your custom certificates.
	// It need to have the following keys: tls.key, tls.crt and optionally ca.crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	CertificateSecretRef *corev1.LocalObjectReference `json:"certificateSecretRef,omitempty"`

	// ValidityDays is the number of days that certificates are valid
	// Default to 365
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=365
	ValidityDays *int `json:"validityDays,omitempty"`

	// RenewalDays is the number of days before certificate expire to become effective renewal
	// Default to 30
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=30
	RenewalDays *int `json:"renewalDays,omitempty"`

	// KeySize is the key size when generate privates keys
	// Default to 2048
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=2048
	KeySize *int `json:"keySize,omitempty"`
}

type ElasticsearchSelfSignedCertificateSpec struct {

	// AltIps permit to set subject alt names of type ip when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltIps []string `json:"altIPs,omitempty"`

	// AltNames permit to set subject alt names of type dns when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltNames []string `json:"altNames,omitempty"`
}

type ElasticsearchIngressSpec struct {

	// Enabled permit to enabled / disabled ingress
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// TargetNodeGroupName permit to define if specific node group is responsible to receive external access, like ingest nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroupName string `json:"targetNodeGroupName,omitempty"`

	// Host is the hostname to access on Elasticsearch
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Host string `json:"host"`

	// SecretRef is the secret ref that store certificates
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// Labels to set in ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to set in ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// IngressSpec it merge with expected ingress spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IngressSpec *networkingv1.IngressSpec `json:"ingressSpec,omitempty"`
}

type ElasticsearchGlobalNodeGroupSpec struct {

	// AdditionalVolumes permit to use additionnal volumes
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdditionalVolumes []ElasticsearchVolumeSpec `json:"additionalVolumes,omitempty"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *ElasticsearchAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// PodDisruptionBudget is the pod disruption budget policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudgetSpec *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// InitContainerResources permit to set resources on init containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	InitContainerResources *corev1.ResourceRequirements `json:"initContainerResources,omitempty"`

	// PodTemplate is merged with expected pod
	// It usefull to add some extra properties on pod spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +optional
	PodTemplate *corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

	// Jvm permit to set extra option on JVM like memory or proxy to download plugins
	// Becarefull with memory, not forget to set the right ressource on pod
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Jvm string `json:"jvm,omitempty"`

	// Config is the Elasticsearch config dedicated for this node groups like roles
	// The key is the file stored on elasticsearch/config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// KeystoreSecretRef is the secret that store the security settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KeystoreSecretRef *corev1.LocalObjectReference `json:"keystoreSecretRef,omitempty"`

	// Labels permit to set labels on containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations permit to set annotation on containers
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
}

type ElasticsearchNodeGroupSpec struct {

	// Name is the the node group name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Replicas is the number of replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas int32 `json:"replicas"`

	// Roles is the list of Elasticsearch roles
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Roles []string `json:"roles"`

	// Persistence is the spec to persist data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *ElasticsearchPersistenceSpec `json:"persistence,omitempty"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *ElasticsearchAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// Resources permit to set ressources on Elasticsearch container
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Jvm permit to set extra option on JVM like Xmx, Xms
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Jvm string `json:"jvm,omitempty"`

	// Config is the Elasticsearch config dedicated for this node groups
	// The key is the file stored on elasticsearch/config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

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

	// Annotations permit to set annotation on containers
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

	// PodTemplate is merged with expected pod
	// It usefull to add some extra properties on pod spec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodTemplate *corev1.PodTemplateSpec `json:"podTemplate,omitempty"`

	// PodDisruptionBudget is the pod disruption budget policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudgetSpec *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// WaitClusterStatus permit to wait the cluster state on readyness probe
	// Default to green
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=green
	WaitClusterStatus string `json:"waitClusterStatus,omitempty"`
}

type ElasticsearchPersistenceSpec struct {
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

type ElasticsearchVolumeSpec struct {

	// Name is the volume name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	corev1.VolumeMount `json:",inline"`

	corev1.VolumeSource `json:",inline"`
}

type ElasticsearchAntiAffinitySpec struct {

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

// ElasticsearchStatus defines the observed state of Elasticsearch
type ElasticsearchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the current cluster deployment phase
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Phase string `json:"phase"`

	// IsError is true if controller is stuck on Error
	// +operator-sdk:csv:customresourcedefinitions:type=status
	IsError *bool `json:"isOnError"`

	// Url is the Elasticsearch endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Url string `json:"url"`

	// CredentialsRef is the secret that store the credentials to access on Elasticsearch
	// +operator-sdk:csv:customresourcedefinitions:type=status
	CredentialsRef corev1.LocalObjectReference `json:"credentialsRef"`

	// Health is the cluster health
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Health string `json:"health"`

	// List of conditions
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Elasticsearch is the Schema for the elasticsearchs API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Deployment,apps/v1},{StatefulSet,apps/v1},{License,elasticsearchapi.k8s.webcenter.fr/v1alpha1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1},{PodMonitor,monitoring.coreos.com/v1},{User,elasticsearchapi.k8s.webcenter.fr/v1alpha1},{Metricbeat,beat.k8s.webcenter.fr/v1alpha1}}
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="CredentialsRef",type="string",JSONPath=".status.credentialsRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Cluster deployment status"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Health",type="string",JSONPath=".status.health",description="Cluster health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Elasticsearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticsearchSpec   `json:"spec,omitempty"`
	Status ElasticsearchStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ElasticsearchList contains a list of Elasticsearch
type ElasticsearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Elasticsearch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Elasticsearch{}, &ElasticsearchList{})
}
