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
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	FilebeatAnnotationKey = "filebeat.k8s.webcenter.fr"
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
	LogstashRef FilebeatLogstashRef `json:"logstashRef,omitempty"`

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
	Deployment FilebeatDeploymentSpec `json:"deployment,omitempty"`

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

	// Routes permit to declare some routes
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Routes []shared.Route `json:"routes,omitempty"`

	// Services permit to declare some services
	// The name is decorated with cluster name and so on
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Services []shared.Service `json:"services,omitempty"`

	// Pki permit to manage certificates you can use for filebeat inputs
	// It will mount them on /usr/share/filebeat/certs/
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Pki FilebeatPkiSpec `json:"pki,omitempty"`
}

type FilebeatPkiSpec struct {
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
	Tls map[string]shared.TlsSelfSignedCertificateSpec `json:"tls,omitempty"`
}

type FilebeatLogstashRef struct {
	// ManagedLogstashRef is the managed Logstash instance by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedLogstashRef *FilebeatLogstashManagedRef `json:"managed,omitempty"`

	// ExternalLogstahsRef is the external Logstash instance not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalLogstashRef *FilebeatLogstashExternalRef `json:"external,omitempty"`

	// LogstashCaSecretRef is the secret that store your custom CA certificates to connect on Logstash via beat protocole.
	// It will add all entry that finish by *.crt or *.pem
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LogstashCaSecretRef *corev1.LocalObjectReference `json:"logstashCASecretRef,omitempty"`
}

type FilebeatLogstashManagedRef struct {
	// Name is the Logstash cluster deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

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
	Port int64 `json:"port"`
}

type FilebeatLogstashExternalRef struct {
	// Addresses is the list of Logstash addresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Addresses []string `json:"addresses"`
}

type FilebeatDeploymentSpec struct {
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

	// Ports is the list of container port to affect on filebeat container
	// It can be usefull to expose syslog input
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`
}

// FilebeatStatus defines the observed state of Filebeat
type FilebeatStatus struct {
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

// Filebeat is the Schema for the filebeats API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{StatefulSet,apps/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1},{ServiceAccount,v1},{RoleBinding,rbac.authorization.k8s.io/v1},{Route,route.openshift.io/v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Certs",type="string",JSONPath=".status.certSecret",description="secret ref that store certs"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
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
