package shared

import (
	corev1 "k8s.io/api/core/v1"
)

// Generic type that represent Image spec
type ImageSpec struct {

	// Image is the image to use when deploy Elasticsearch
	// It can be usefull to use internal registry or mirror
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the image pull policy to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is the image pull secrets to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}
