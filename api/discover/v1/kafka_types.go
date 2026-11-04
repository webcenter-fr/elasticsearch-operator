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
	KafkaAnnotationKey = "kafka.discover.k8s.webcenter.fr"
)

// KafkaSpec defines the desired state of Kafka.
type KafkaSpec struct {
	Discover `json:",inline"`

	KafkaRef KafkaRef `json:"kafkaRef"`
}

// KafkaRef is the Kafka reference to connect on
type KafkaRef struct {
	// ManagedKafkaRef is the managed Kafka instance by Strimzi operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ManagedKafkaRef *KafkaManagedRef `json:"managed,omitempty"`

	// ExternalKafkaRef is the external Kafka instance not managed by Strimzi operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExternalKafkaRef *KafkaExternalRef `json:"external,omitempty"`

	// KafkaCaSecretRef is the secret that store your custom CA certificates to connect on Kafka.
	// It will add all entry that finish by *.crt or *.pem
	// If Kafka is managed and use internal PKI no need to set it. I will use it by design
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	KafkaCaSecretRef *corev1.LocalObjectReference `json:"kafkaCASecretRef,omitempty"`
}

// KafkaManagedRef is the Kafka reference to connect on a Kafka cluster managed by Strimzi operator
type KafkaManagedRef struct {
	// Name is the Kafka cluster deployed by operator
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// Namespace is the namespace where Kafka is deployed by operator
	// No need to set if is deployed on the same namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// TargetListener is the target listener that expose Kafka cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetListener string `json:"targetListener,omitempty"`

	// UserRef is the user reference to connect on Kafka.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	UserRef *corev1.LocalObjectReference `json:"userRef"`
}

// KafkaExternalRef is the Kafka reference to connect on a Kafka cluster not managed by Strimzi operator
type KafkaExternalRef struct {
	// Addresses is the list of Kafka addresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Addresses []string `json:"addresses"`

	// UserSecretRef is the secret that store your user SSL certificate / key.
	// It must have 2 entry:
	//   - user.crt: is the user certificate
	//   - user.key: is the user private key
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	UserSecretRef *corev1.LocalObjectReference `json:"userSecretRef"`
}

// KafkaStatus defines the observed state of Kafka.
type KafkaStatus struct {
	DiscoverStatus `json:",inline"`

	multiphase.DefaultMultiPhaseObjectStatus `json:",inline"`

	// KafkaUserSecretRef is the secret that store all information to connect on Kafka cluster
	// +operator-sdk:csv:customresourcedefinitions:type=status
	KafkaUserSecretRef *string `json:"kafkaUserSecretRef,omitempty"`

	// KafkaCASecretRef is the secret that store CA certificate to connect on Kafka cluster
	// +operator-sdk:csv:customresourcedefinitions:type=status
	KafkaCASecretRef *string `json:"kafkaCASecretRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:storageversion

// Kafka is the Schema for the kafkas API.
// It will generate and maintain secret that store all information to connect on Kafka cluster
// Ca certificate, Ca trustore, Ca trustore password, user certificate, user key,  user keystore, user keystore password, user trustore, user trustore password and bootstrap URL
// +operator-sdk:csv:customresourcedefinitions:resources={{Secret,v1}}
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase"
// +kubebuilder:printcolumn:name="secretFile",type="string",JSONPath=".status.secretFileRef",description="secret ref that store certificate files"
// +kubebuilder:printcolumn:name="secretEnv",type="string",JSONPath=".status.secretEnvRef",description="secret ref that store env variables"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec,omitempty"`
	Status KafkaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaList contains a list of Kafka.
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}
