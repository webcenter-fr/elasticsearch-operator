package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCAElasticsearchSecret(t *testing.T) {
	var (
		err      error
		s        []corev1.Secret
		o        *beatcrd.Filebeat
		esSecret *corev1.Secret
	)

	// When no elasticsearch ref
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}
	s, err = buildCAElasticsearchSecrets(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, s)

	// With default values
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	labels := map[string]string{
		"cluster":                   "test",
		"filebeat.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"filebeat.k8s.webcenter.fr": "true",
	}

	esSecret = &corev1.Secret{
		Data: map[string][]byte{
			"ca.crt": []byte("certificate"),
		},
	}

	s, err = buildCAElasticsearchSecrets(o, esSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, s)
	assert.Equal(t, "test-ca-es-fb", s[0].Name)
	assert.Equal(t, "default", s[0].Namespace)
	assert.Equal(t, labels, s[0].Labels)
	assert.Equal(t, annotations, s[0].Annotations)
	assert.Equal(t, []byte("certificate"), s[0].Data["ca.crt"])
}
