package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err      error
		s        *corev1.Secret
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
	s, err = BuildCredentialSecret(o, nil)
	assert.NoError(t, err)
	assert.Nil(t, s)

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
			"beats_system": []byte("password"),
		},
	}

	s, err = BuildCredentialSecret(o, esSecret)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, "test-credential-fb", s.Name)
	assert.Equal(t, "default", s.Namespace)
	assert.Equal(t, labels, s.Labels)
	assert.Equal(t, annotations, s.Annotations)
	assert.Equal(t, []byte("password"), s.Data["beats_system"])

}
