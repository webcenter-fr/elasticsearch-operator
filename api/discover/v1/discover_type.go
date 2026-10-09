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
	corev1 "k8s.io/api/core/v1"
)

const (
	DiscoverAnnotationKey                  = "discover.k8s.webcenter.fr"
	DiscoverTypeKafka         DiscoverType = "kafka"
	DiscoverTypeLogstash      DiscoverType = "logstash"
	DiscoverTypeElasticsearch DiscoverType = "elasticsearch"
)

type DiscoverType string

// Discover defines the global fields for all discover types
type Discover struct {
	// TargetSecretFileRef defines the secret where to store discovered information as files
	// You can find here certificate, private key, keystore, truststore ...
	// Default to the name of the Discover resource with suffix "-file"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetSecretFileRef *corev1.LocalObjectReference `json:"targetSecretRef,omitempty"`

	// TargetSecretEnvRef defines the secret where to store discovered information as env variables
	// You can find here bootstrap servers, user, password ...
	// Default to the name of the Discover resource with suffix "-env"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	TargetSecretEnvRef *corev1.LocalObjectReference `json:"targetSecretEnvRef,omitempty"`

	// Name is the logical name
	// it used to compute the environment key suffix on secret targetSecretEnvRef
	// it used to compute the mount path of secret targetSecretFileRef
	// Default to the name of the Discover resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name *string `json:"name,omitempty"`
}

// DiscoverStatus defines the observed state of Discover
type DiscoverStatus struct {
	// SecretFileRef is the name of the secret where file-based information is stored
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	SecretFileRef *string `json:"secretFileRef,omitempty"`

	// SecretEnvRef is the name of the secret where env-based information is stored
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	SecretEnvRef *string `json:"secretEnvRef,omitempty"`
}

// DiscoverRef permit to reference an existing discover
type DiscoverRef struct {
	// Kafka permit to reference a Kafka discover
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Kafka *corev1.LocalObjectReference `json:"kafka,omitempty"`

	// Elasticsearch permit to reference an Elasticsearch discover
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Elasticsearch *corev1.LocalObjectReference `json:"elasticsearch,omitempty"`

	// Logstash permit to reference a Logstash discover
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Logstash *corev1.LocalObjectReference `json:"logstash,omitempty"`
}
