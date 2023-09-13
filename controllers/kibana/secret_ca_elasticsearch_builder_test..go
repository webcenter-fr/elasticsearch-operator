package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCAElasticsearchSecret(t *testing.T) {
	var (
		err      error
		secrets  []corev1.Secret
		o        *kibanacrd.Kibana
		esSecret *corev1.Secret
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	labels := map[string]string{
		"cluster":                 "test",
		"kibana.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"kibana.k8s.webcenter.fr": "true",
	}

	esSecret = &corev1.Secret{
		Data: map[string][]byte{
			"ca.crt": []byte("certificate"),
		},
	}

	secrets, err = buildCAElasticsearchSecrets(o, esSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Equal(t, "test-ca-es-kb", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, labels, secrets[0].Labels)
	assert.Equal(t, annotations, secrets[0].Annotations)
	assert.Equal(t, []byte("certificate"), secrets[0].Data["ca.crt"])

}
