package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServices(t *testing.T) {
	var (
		err      error
		services []corev1.Service
		o        *elasticsearchcrd.Elasticsearch
	)
	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(services))
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_default.yaml", &services[0], scheme.Scheme)

	// When nodes Groups
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
				},
			},
		},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(services))
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_master.yaml", &services[1], scheme.Scheme)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_master_headless.yaml", &services[2], scheme.Scheme)
}
