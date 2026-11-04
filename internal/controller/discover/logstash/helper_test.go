package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretNameForFile(t *testing.T) {
	// Test with custom target secret file ref
	logstash := &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-logstash",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "custom-file-secret",
				},
			},
		},
	}

	result := GetSecretNameForFile(logstash)
	assert.Equal(t, "custom-file-secret", result)

	// Test with nil target secret file ref
	logstash.Spec.TargetSecretFileRef = nil
	result = GetSecretNameForFile(logstash)
	assert.Equal(t, "test-logstash-file", result)

	// Test with empty name in target secret file ref
	logstash.Spec.TargetSecretFileRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForFile(logstash)
	assert.Equal(t, "test-logstash-file", result)
}

func TestGetSecretNameForEnv(t *testing.T) {
	// Test with custom target secret env ref
	logstash := &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-logstash",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "custom-env-secret",
				},
			},
		},
	}

	result := GetSecretNameForEnv(logstash)
	assert.Equal(t, "custom-env-secret", result)

	// Test with nil target secret env ref
	logstash.Spec.TargetSecretEnvRef = nil
	result = GetSecretNameForEnv(logstash)
	assert.Equal(t, "test-logstash-env", result)

	// Test with empty name in target secret env ref
	logstash.Spec.TargetSecretEnvRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForEnv(logstash)
	assert.Equal(t, "test-logstash-env", result)
}
