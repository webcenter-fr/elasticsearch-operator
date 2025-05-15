package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCALogstashSecret(t *testing.T) {
	var (
		err      error
		secrets  []*corev1.Secret
		o        *beatcrd.Filebeat
		lsSecret *corev1.Secret
	)

	// When no logstash ref
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	secrets, err = buildCALogstashSecrets(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, secrets)

	// When logstash is managed and logstash PKI is enabled
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

	lsSecret = &corev1.Secret{
		Data: map[string][]byte{
			"ca.crt": []byte("certificate"),
		},
	}

	secrets, err = buildCALogstashSecrets(o, lsSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Equal(t, "test-ca-ls-fb", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, labels, secrets[0].Labels)
	assert.Equal(t, annotations, secrets[0].Annotations)
	assert.Equal(t, []byte("certificate"), secrets[0].Data["ca.crt"])
}
