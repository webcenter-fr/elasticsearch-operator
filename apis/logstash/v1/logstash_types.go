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

const (
	LogstashAnnotationKey = "logstash.k8s.webcenter.fr"
)

// LogstashSpec defines the desired state of Logstash
// +k8s:openapi-gen=true
type LogstashSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// It will expose CA certificate and Elasticsearch URL as encironment variable to use it in logstash setting
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef,omitempty"`

	// Version is the logstash version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// PluginsList is the list of additionnal plugin to install on each logstash instance
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PluginsList []string `json:"pluginsList,omitempty"`

	// Config is the Logstash config
	// The key is the file stored on logstash/config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Pipeline is the pipeline specification
	// The key is the file stored on logstash/pipelines
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Pipeline map[string]string `json:"pipeline,omitempty"`

	// Patterns is the patterns specification used by grok
	// The key is the file stored on logstash/patterns
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Pattern map[string]string `json:"pattern,omitempty"`

	// KeystoreSecretRef is the secret that store the security settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KeystoreSecretRef *corev1.LocalObjectReference `json:"keystoreSecretRef,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Deployment LogstashDeploymentSpec `json:"deployment,omitempty"`

	// Monitoring permit to monitor current cluster
	// Default, it not monitor cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Monitoring shared.MonitoringSpec `json:"monitoring,omitempty"`

	// Ingresses permit to declare some ingresses
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingresses []shared.Ingress `json:"ingresses,omitempty"`

	// Services permit to declare some services
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Services []shared.Service `json:"services,omitempty"`

	// Pki permit to manage certificates you can use for Logstash inputs
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Pki LogstashPkiSpec `json:"pki,omitempty"`
}

type LogstashPkiSpec struct {

	// Enabled permit to enabled the internal PKI
	// Default to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

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

	// Tls is the list of TLS certificates to manage
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Tls map[string]LogstashTlsSpec `json:"tls,omitempty"`
}

type LogstashTlsSpec struct {
	shared.TlsSelfSignedCertificateSpec `json:",inline"`

	// Consumer it the service that will consume certificate
	// It support filebeat, logstash and custom
	// Default to custom
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=custom
	Consumer string `json:"consumer,omitempty"`
}

type LogstashDeploymentSpec struct {
	shared.Deployment `json:",inline"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *shared.DeploymentAntiAffinitySpec `json:"antiAffinity,omitempty"`

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
	AdditionalVolumes []shared.DeploymentVolumeSpec `json:"additionalVolumes,omitempty"`

	// Persistence is the spec to persist data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Persistence *shared.DeploymentPersistenceSpec `json:"persistence,omitempty"`

	// Ports is the list of container port to affect on logstash container
	// It can be usefull to expose beats input
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`
}

// LogstashStatus defines the observed state of Logstash
type LogstashStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`

	// CertSecretName is the secret name that store certs generated for inputs
	// +operator-sdk:csv:customresourcedefinitions:type=status
	CertSecretName string `json:"certSecret,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Logstash is the Schema for the logstashes API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{StatefulSet,apps/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Logstash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogstashSpec   `json:"spec,omitempty"`
	Status LogstashStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LogstashList contains a list of Logstash
type LogstashList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Logstash `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Logstash{}, &LogstashList{})
}
