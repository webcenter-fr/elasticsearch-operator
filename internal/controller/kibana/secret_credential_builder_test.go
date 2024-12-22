package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err      error
		s        []corev1.Secret
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
			"kibana_system":          []byte("password"),
			"remote_monitoring_user": []byte("password"),
		},
	}

	s, err = buildCredentialSecrets(o, esSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, s)
	assert.Equal(t, "test-credential-kb", s[0].Name)
	assert.Equal(t, "default", s[0].Namespace)
	assert.Equal(t, labels, s[0].Labels)
	assert.Equal(t, annotations, s[0].Annotations)
	assert.Equal(t, []byte("password"), s[0].Data["kibana_system"])
	assert.Equal(t, []byte("password"), s[0].Data["remote_monitoring_user"])
	assert.Equal(t, []byte("kibana_system"), s[0].Data["username"])
}
