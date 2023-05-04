package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildUserSystem(t *testing.T) {

	var (
		o     *elasticsearchcrd.Elasticsearch
		s     *corev1.Secret
		users []elasticsearchapicrd.User
	)

	// Normal
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      GetSecretNameForCredentials(o),
		},
		Data: map[string][]byte{
			"kibana_system": []byte("password"),
		},
	}

	users, err := BuildUserSystem(o, s)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/user_kibana.yml", &users[0], test.CleanApi)
}
