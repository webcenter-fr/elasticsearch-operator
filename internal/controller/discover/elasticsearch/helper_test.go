package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretNameForFile(t *testing.T) {
	// Test with custom target secret file ref
	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-elasticsearch",
		},
		Spec: discovercrd.ElasticsearchSpec{
			Discover: discovercrd.Discover{
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "custom-file-secret",
				},
			},
		},
	}

	result := GetSecretNameForFile(o)
	assert.Equal(t, "custom-file-secret", result)

	// Test with nil target secret file ref
	o.Spec.TargetSecretFileRef = nil
	result = GetSecretNameForFile(o)
	assert.Equal(t, "test-elasticsearch-file", result)

	// Test with empty name in target secret file ref
	o.Spec.TargetSecretFileRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForFile(o)
	assert.Equal(t, "test-elasticsearch-file", result)
}

func TestGetSecretNameForEnv(t *testing.T) {
	// Test with custom target secret env ref
	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-elasticsearch",
		},
		Spec: discovercrd.ElasticsearchSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "custom-env-secret",
				},
			},
		},
	}

	result := GetSecretNameForEnv(o)
	assert.Equal(t, "custom-env-secret", result)

	// Test with nil target secret env ref
	o.Spec.TargetSecretEnvRef = nil
	result = GetSecretNameForEnv(o)
	assert.Equal(t, "test-elasticsearch-env", result)

	// Test with empty name in target secret env ref
	o.Spec.TargetSecretEnvRef = &corev1.LocalObjectReference{
		Name: "",
	}
	result = GetSecretNameForEnv(o)
	assert.Equal(t, "test-elasticsearch-env", result)
}
