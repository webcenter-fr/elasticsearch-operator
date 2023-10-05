package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildUserSystem(t *testing.T) {

	var (
		o     *elasticsearchcrd.Elasticsearch
		s     *corev1.Secret
		users []elasticsearchapicrd.User
	)
	sch := scheme.Scheme
	elasticsearchapicrd.AddToScheme(sch)

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

	users, err := buildSystemUsers(o, s)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*elasticsearchapicrd.User](t, "testdata/user_kibana.yml", &users[0], sch)
}
