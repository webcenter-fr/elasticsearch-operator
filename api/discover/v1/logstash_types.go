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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LogstashAnnotationKey = "logstash.discover.k8s.webcenter.fr"
)

// LogstashSpec defines the desired state of Logstash.
type LogstashSpec struct {
	Discover `json:",inline"`

	// LogstashRef is the Logstash ref to connect on.
	// It will generate Logstash output base on it
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LogstashRef LogstashRef `json:"logstashRef,omitempty"`
}

// LogstashRef is the Logstash reference to connect on
type LogstashRef struct {
	// ManagedLogstashRef is the managed Logstash instance by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedLogstashRef *LogstashManagedRef `json:"managed,omitempty"`

	// ExternalLogstahsRef is the external Logstash instance not managed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalLogstashRef *LogstashExternalRef `json:"external,omitempty"`

	// LogstashCaSecretRef is the secret that store your custom CA certificates to connect on Logstash via beat protocole.
	// It will add all entry that finish by *.crt or *.pem
	// If Logstash is managed and use internal PKI no need to set it. I will use it by design
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LogstashCaSecretRef *corev1.LocalObjectReference `json:"logstashCASecretRef,omitempty"`
}

// LogstashManagedRef is the Logstash reference to connect on a Logstash cluster managed by operator
type LogstashManagedRef struct {
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

	// UserRef is the user reference to connect on Logstash.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	UserRef *corev1.SecretReference `json:"userRef"`
}

// LogstashExternalRef is the Logstash reference to connect on a Logstash cluster not managed by operator
type LogstashExternalRef struct {
	// Addresses is the list of Logstash addresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Addresses []string `json:"addresses"`

	// UserSecretRef is the secret that store your user SSL certificate / key.
	// It must have 2 entry:
	//   - user.key: the user private key
	//   - user.crt: the user certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	UserSecretRef *corev1.LocalObjectReference `json:"userSecretRef"`
}

// LogstashStatus defines the observed state of Logstash.
type LogstashStatus struct {
	DiscoverStatus `json:",inline"`

	multiphase.DefaultMultiPhaseObjectStatus `json:",inline"`

	// LogstashCASecretRef is the secret that store CA certificate to connect on Logstash
	// +operator-sdk:csv:customresourcedefinitions:type=status
	LogstashCASecretRef *string `json:"logstashCASecretRef,omitempty"`

	// LogstashUserSecretRef is the secret that store user certificates to connect on Logstash
	// +operator-sdk:csv:customresourcedefinitions:type=status
	LogstashUserSecretRef *string `json:"logstashUserSecretRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:storageversion

// Logstash is the Schema for the logstashes API.
// It will generate and maintain secret that store CA certificate for Logstash.
// It will also maintain the URL to connect on Logstash.
// +operator-sdk:csv:customresourcedefinitions:resources={{Secret,v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="secretFile",type="string",JSONPath=".status.secretFileRef",description="secret ref that store certificate files"
// +kubebuilder:printcolumn:name="secretEnv",type="string",JSONPath=".status.secretEnvRef",description="secret ref that store env variables"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Logstash struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogstashSpec   `json:"spec,omitempty"`
	Status LogstashStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LogstashList contains a list of Logstash.
type LogstashList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Logstash `json:"items"`
}
