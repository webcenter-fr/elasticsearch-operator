package metricbeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err      error
		secrets  []*corev1.Secret
		o        *beatcrd.Metricbeat
		esSecret *corev1.Secret
	)

	// When no elasticsearch ref
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}
	secrets, err = buildCredentialSecrets(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, secrets)

	// With default values
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	labels := map[string]string{
		"cluster":                     "test",
		"metricbeat.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"metricbeat.k8s.webcenter.fr": "true",
	}

	esSecret = &corev1.Secret{
		Data: map[string][]byte{
			"remote_monitoring_user": []byte("password"),
		},
	}

	secrets, err = buildCredentialSecrets(o, esSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Equal(t, "test-credential-mb", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, labels, secrets[0].Labels)
	assert.Equal(t, annotations, secrets[0].Annotations)
	assert.Equal(t, []byte("password"), secrets[0].Data["remote_monitoring_user"])
}
