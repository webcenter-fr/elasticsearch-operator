package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err      error
		s        *corev1.Secret
		o        *logstashcrd.Logstash
		esSecret *corev1.Secret
	)

	// When no Elasticsearch ref
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	s, err = BuildCredentialSecret(o, nil)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	labels := map[string]string{
		"cluster":                   "test",
		"logstash.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"logstash.k8s.webcenter.fr": "true",
	}

	esSecret = &corev1.Secret{
		Data: map[string][]byte{
			"logstash_system":        []byte("password"),
			"remote_monitoring_user": []byte("password"),
		},
	}

	s, err = BuildCredentialSecret(o, esSecret)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, "test-credential-ls", s.Name)
	assert.Equal(t, "default", s.Namespace)
	assert.Equal(t, labels, s.Labels)
	assert.Equal(t, annotations, s.Annotations)
	assert.Equal(t, []byte("password"), s.Data["logstash_system"])
	assert.Equal(t, []byte("password"), s.Data["remote_monitoring_user"])

}
