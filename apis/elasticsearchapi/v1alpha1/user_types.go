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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UserSpec defines the desired state of User
// +k8s:openapi-gen=true
type UserSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRefSpec is the Elasticsearch cluster reference
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRefSpec ElasticsearchRefSpec `json:"elasticsearchRef"`

	// Enabled permit to enable user
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Enabled bool `json:"enabled,omitempty"`

	// Username is the user name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Username string `json:"username,omitempty"`

	// Email is the email user
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Email string `json:"email,omitempty"`

	// FullName is the full name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	FullName string `json:"fullName,omitempty"`

	// Metadata is the meta data
	// Is JSON string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Metadata string `json:"metadata,omitempty"`

	// CredentialSecretRef permit to set password. Or you can use password hash
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SecretRef *UserSecret `json:"secretRef,omitempty"`

	// PasswordHash is the password as hash
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PasswordHash string `json:"passwordHash,omitempty"`

	// Roles is the list of roles
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Roles []string `json:"roles,omitempty"`

	// IsProtected must be set when you manage protected account like kibana_system
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IsProtected *bool `json:"isProtected,omitempty"`
}

type UserSecret struct {

	// Name is the secret name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`

	// key is the key name on secret to read the effective password
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Key string `json:"key"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions"`

	PasswordHash string `json:"passwordHash"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
