package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMap(t *testing.T) {

	var (
		o         *cerebrocrd.Cerebro
		esList    []elasticsearchcrd.Elasticsearch
		err       error
		configMap *corev1.ConfigMap
	)

	// When no target elasticsearch
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
			Labels: map[string]string{
				"label1": "value1",
			},
			Annotations: map[string]string{
				"anno1": "value1",
			},
		},
		Spec: cerebrocrd.CerebroSpec{
			Config: map[string]string{
				"application.conf": "test = test\n",
				"log4j.yml":        "log.test: test\n",
			},
		},
	}

	configMap, err = BuildConfigMap(o, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_default.yml", configMap, test.CleanApi)

	// When some elasticsearch targets
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	esList = []elasticsearchcrd.Elasticsearch{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "es1",
				Namespace: "default",
			},
			Spec: elasticsearchcrd.ElasticsearchSpec{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "es2",
				Namespace: "default",
			},
			Spec: elasticsearchcrd.ElasticsearchSpec{
				ClusterName: "test2",
			},
		},
	}

	configMap, err = BuildConfigMap(o, esList)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_elasticsearch_targets.yml", configMap, test.CleanApi)
}
