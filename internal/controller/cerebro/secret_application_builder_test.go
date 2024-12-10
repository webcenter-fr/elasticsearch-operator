package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildApplicationSecret(t *testing.T) {
	var (
		err error
		s   []corev1.Secret
		o   *cerebrocrd.Cerebro
	)

	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	labels := map[string]string{
		"cluster":                  "test",
		"cerebro.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"cerebro.k8s.webcenter.fr": "true",
	}

	s, err = buildApplicationSecrets(o)
	assert.NoError(t, err)
	assert.NotEmpty(t, s)
	assert.Equal(t, "test-application-cb", s[0].Name)
	assert.Equal(t, "default", s[0].Namespace)
	assert.Equal(t, labels, s[0].Labels)
	assert.Equal(t, annotations, s[0].Annotations)
	assert.NotEmpty(t, s[0].Data["application"])
}
