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
	Tls shared.TlsSpec `json:"tls,omitempty"`

	// LicenseSecretRef permit to set secret that contain Elasticsearch license on key `license`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LicenseSecretRef *corev1.LocalObjectReference `json:"licenseSecretRef,omitempty"`

	// Monitoring permit to monitor current cluster
	// Default, it not monitor cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Monitoring shared.MonitoringSpec `json:"monitoring,omitempty"`
}

type ElasticsearchEndpointSpec struct {
	// Ingress permit to set ingress settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingress *ElasticsearchIngressSpec `json:"ingress,omitempty"`

	// Route permit to set Openshift route settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Route *ElasticsearchRouteSpec `json:"route,omitempty"`

	// Load balancer permit to set load balancer settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LoadBalancer *ElasticsearchLoadBalancerSpec `json:"loadBalancer,omitempty"`
}

type ElasticsearchLoadBalancerSpec struct {
	shared.EndpointLoadBalancerSpec `json:",inline"`

	// TargetNodeGroupName permit to define if specific node group is responsible to receive external access, like ingest nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroupName string `json:"targetNodeGroupName,omitempty"`
}

type ElasticsearchIngressSpec struct {
	shared.EndpointIngressSpec `json:",inline"`

	// TargetNodeGroupName permit to define if specific node group is responsible to receive external access, like ingest nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroupName string `json:"targetNodeGroupName,omitempty"`
}

type ElasticsearchRouteSpec struct {
	shared.EndpointRouteSpec `json:",inline"`

	// TargetNodeGroupName permit to define if specific node group is responsible to receive external access, like ingest nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetNodeGroupName string `json:"targetNodeGroupName,omitempty"`
}

type ElasticsearchGlobalNodeGroupSpec struct {
	// AdditionalVolumes permit to use additionnal volumes
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdditionalVolumes []shared.DeploymentVolumeSpec `json:"additionalVolumes,omitempty"`

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

	// PodTemplate is merged with expected pod
	// It usefull to add some extra properties on pod spec
	// @clean
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
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

	// CacertsSecretRef is the secret that store custom CA to import on cacerts
	// It usefull to access on onpremise S3 service to store snaoshot and more
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	CacertsSecretRef *corev1.LocalObjectReference `json:"caSecretRef,omitempty"`

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
	shared.Deployment `json:",inline"`

	// Name is the the node group name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Roles is the list of Elasticsearch roles
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Roles []string `json:"roles"`

	// Persistence is the spec to persist data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *shared.DeploymentPersistenceSpec `json:"persistence,omitempty"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *shared.DeploymentAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// Jvm permit to set extra option on JVM like Xmx, Xms
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Jvm string `json:"jvm,omitempty"`

	// Config is the Elasticsearch config dedicated for this node groups
	// The key is the file stored on elasticsearch/config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// PodDisruptionBudget is the pod disruption budget policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudgetSpec *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`

	// WaitClusterStatus permit to wait the cluster state on readyness probe
	// Default to green
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=green
	// +kubebuilder:validation:Enum=green;yellow;red
	WaitClusterStatus string `json:"waitClusterStatus,omitempty"`
}

// ElasticsearchStatus defines the observed state of Elasticsearch
type ElasticsearchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`

	// IsBootstrapping is true when the cluster is bootstraping
	// +operator-sdk:csv:customresourcedefinitions:type=status
	IsBootstrapping *bool `json:"isBootstrapping,omitempty"`

	// Url is the Elasticsearch endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Url string `json:"url,omitempty"`

	// CredentialsRef is the secret that store the credentials to access on Elasticsearch
	// +operator-sdk:csv:customresourcedefinitions:type=status
	CredentialsRef corev1.LocalObjectReference `json:"credentialsRef,omitempty"`

	// Health is the cluster health
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Health string `json:"health,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Elasticsearch is the Schema for the elasticsearchs API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Deployment,apps/v1},{StatefulSet,apps/v1},{License,elasticsearchapi.k8s.webcenter.fr/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1},{PodMonitor,monitoring.coreos.com/v1},{User,elasticsearchapi.k8s.webcenter.fr/v1},{Metricbeat,beat.k8s.webcenter.fr/v1},{ServiceAccount,v1},{RoleBinding,rbac.authorization.k8s.io/v1},{Route,route.openshift.io/v1}}
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="CredentialsRef",type="string",JSONPath=".status.credentialsRef.name"
// +kubebuilder:printcolumn:name="Health",type="string",JSONPath=".status.health",description="Cluster health"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
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
