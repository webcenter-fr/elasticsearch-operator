package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCredentialSecret(t *testing.T) {
	var (
		err error
		s   []corev1.Secret
		o   *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	labels := map[string]string{
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	s, err = buildCredentialSecrets(o)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, "test-credential-es", s[0].Name)
	assert.Equal(t, "default", s[0].Namespace)
	assert.Equal(t, labels, s[0].Labels)
	assert.Equal(t, annotations, s[0].Annotations)
	assert.NotEmpty(t, s[0].Data["elastic"])
	assert.NotEmpty(t, s[0].Data["kibana_system"])
	assert.NotEmpty(t, s[0].Data["logstash_system"])
	assert.NotEmpty(t, s[0].Data["beats_system"])
	assert.NotEmpty(t, s[0].Data["apm_system"])
	assert.NotEmpty(t, s[0].Data["remote_monitoring_user"])

}
