package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildServices(t *testing.T) {

	var (
		err      error
		services []corev1.Service
		o        *elasticsearchapi.Elasticsearch
	)
	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{},
	}

	services, err = BuildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(services))
	test.EqualFromYamlFile(t, "testdata/service_default.yaml", &services[0], test.CleanApi)

	// When nodes Groups
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
				},
			},
		},
	}

	services, err = BuildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(services))
	test.EqualFromYamlFile(t, "testdata/service_master.yaml", &services[1], test.CleanApi)
	test.EqualFromYamlFile(t, "testdata/service_master_headless.yaml", &services[2], test.CleanApi)

}
