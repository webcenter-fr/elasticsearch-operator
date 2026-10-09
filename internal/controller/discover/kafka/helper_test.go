package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretNameForFile(t *testing.T) {
	// Test with custom target secret file ref
	kafka := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-kafka",
		},
		Spec: discovercrd.KafkaSpec{
			Discover: discovercrd.Discover{
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "custom-file-secret",
				},
			},
		},
	}

	result := GetSecretNameForFile(kafka)
	assert.Equal(t, "custom-file-secret", result)

	// Test with nil target secret file ref
	kafka.Spec.TargetSecretFileRef = nil
	result = GetSecretNameForFile(kafka)
	assert.Equal(t, "test-kafka-file", result)

	// Test with empty name in target secret file ref
	kafka.Spec.TargetSecretFileRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForFile(kafka)
	assert.Equal(t, "test-kafka-file", result)
}

func TestGetSecretNameForEnv(t *testing.T) {
	// Test with custom target secret env ref
	kafka := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-kafka",
		},
		Spec: discovercrd.KafkaSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "custom-env-secret",
				},
			},
		},
	}

	result := GetSecretNameForEnv(kafka)
	assert.Equal(t, "custom-env-secret", result)

	// Test with nil target secret env ref
	kafka.Spec.TargetSecretEnvRef = nil
	result = GetSecretNameForEnv(kafka)
	assert.Equal(t, "test-kafka-env", result)

	// Test with empty name in target secret env ref
	kafka.Spec.TargetSecretEnvRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForEnv(kafka)
	assert.Equal(t, "test-kafka-env", result)
}
