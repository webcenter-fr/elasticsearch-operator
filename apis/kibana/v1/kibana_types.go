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
limitations under the License.true
*/

package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	KibanaAnnotationKey = "kibana.k8s.webcenter.fr"
)

// KibanaSpec defines the desired state of Kibana
// +k8s:openapi-gen=true
type KibanaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	shared.ImageSpec `json:",inline"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Version is the Kibana version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// PluginsList is the list of additionnal plugin to install on each Kibana instance
	// Default is empty
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PluginsList []string `json:"pluginsList,omitempty"`

	// Endpoint permit to set endpoints to access on Kibana from external kubernetes
	// You can set ingress and / or load balancer
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Endpoint KibanaEndpointSpec `json:"endpoint,omitempty"`

	// Config is the Kibana config
	// The key is the file stored on kibana/config
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// KeystoreSecretRef is the secret that store the security settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KeystoreSecretRef *corev1.LocalObjectReference `json:"keystoreSecretRef,omitempty"`

	// Tls permit to set the TLS setting for Kibana access
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Tls KibanaTlsSpec `json:"tls,omitempty"`

	// Deployment permit to set the deployment settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Deployment KibanaDeploymentSpec `json:"deployment,omitempty"`

	// Monitoring permit to monitor current cluster
	// Default, it not monitor cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Monitoring KibanaMonitoringSpec `json:"monitoring,omitempty"`
}

type KibanaMonitoringSpec struct {

	// Prometheus permit to monitor cluster with Prometheus and graphana (via exporter)
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Prometheus *KibanaPrometheusSpec `json:"prometheus,omitempty"`

	// Metricbeat permit to monitor cluster with metricbeat and to dedicated monitoring cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Metricbeat *shared.MetricbeatMonitoringSpec `json:"metricbeat,omitempty"`
}

type KibanaPrometheusSpec struct {

	// Enabled permit to enable Prometheus monitoring
	// It will deploy exporter for Kibana and add podMonitor policy
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

type KibanaEndpointSpec struct {
	// Ingress permit to set ingress settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ingress *KibanaIngressSpec `json:"ingress,omitempty"`

	// Load balancer permit to set load balancer settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LoadBalancer *KibanaLoadBalancerSpec `json:"loadBalancer,omitempty"`
}

type KibanaLoadBalancerSpec struct {
	// Enabled permit to enabled / disabled load balancer
	// Cloud provider need to support it
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`
}

type KibanaIngressSpec struct {

	// Enabled permit to enabled / disabled ingress
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Host is the hostname to access on Kibana
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

type KibanaTlsSpec struct {

	// Enabled permit to enabled TLS on Kibana
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// SelfSignedCertificate permit to set self signed certificate settings
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SelfSignedCertificate *KibanaSelfSignedCertificateSpec `json:"selfSignedCertificate,omitempty"`

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

type KibanaSelfSignedCertificateSpec struct {

	// AltIps permit to set subject alt names of type ip when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltIps []string `json:"altIPs,omitempty"`

	// AltNames permit to set subject alt names of type dns when generate certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AltNames []string `json:"altNames,omitempty"`
}

type KibanaDeploymentSpec struct {
	// Replicas is the number of replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replicas int32 `json:"replicas"`

	// AntiAffinity permit to set anti affinity policy
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AntiAffinity *KibanaAntiAffinitySpec `json:"antiAffinity,omitempty"`

	// Resources permit to set ressources on Kibana container
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
	Node string `json:"node,omitempty"`

	// InitContainerResources permit to set resources on init containers
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	InitContainerResources *corev1.ResourceRequirements `json:"initContainerResources,omitempty"`
}

type KibanaAntiAffinitySpec struct {

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

// KibanaStatus defines the observed state of Kibana
type KibanaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicMultiPhaseObjectStatus `json:",inline"`

	// Url is the Kibana endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Url string `json:"url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Kibana is the Schema for the kibanas API
// +operator-sdk:csv:customresourcedefinitions:resources={{Ingress,networking.k8s.io/v1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Deployment,apps/v1},{NetworkPolicy,networking.k8s.io/v1},{PodDisruptionBudget,policy/v1},{PodMonitor,monitoring.coreos.com/v1}}
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Kibana struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KibanaSpec   `json:"spec,omitempty"`
	Status KibanaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KibanaList contains a list of Kibana
type KibanaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kibana `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kibana{}, &KibanaList{})
}
