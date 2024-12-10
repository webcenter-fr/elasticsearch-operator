package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err      error
		secrets  []corev1.Secret
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

	secrets, err = buildCredentialSecrets(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, secrets)

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

	secrets, err = buildCredentialSecrets(o, esSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Equal(t, "test-credential-ls", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, labels, secrets[0].Labels)
	assert.Equal(t, annotations, secrets[0].Annotations)
	assert.Equal(t, []byte("password"), secrets[0].Data["logstash_system"])
	assert.Equal(t, []byte("password"), secrets[0].Data["remote_monitoring_user"])
}
