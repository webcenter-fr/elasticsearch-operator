package cerebro

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildConfigMap(t *testing.T) {
	var (
		o          *cerebrocrd.Cerebro
		esList     []elasticsearchcrd.Elasticsearch
		err        error
		configMaps []*corev1.ConfigMap
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
			Config: ptr.To(`test2 = test2`),
			ExtraConfigs: map[string]string{
				"application.conf": "test = test\n",
				"log4j.yml":        "log.test: test\n",
			},
		},
	}

	configMaps, err = buildConfigMaps(o, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default.yml", configMaps[0], scheme.Scheme)

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

	configMaps, err = buildConfigMaps(o, esList, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_elasticsearch_targets.yml", configMaps[0], scheme.Scheme)

	// When some external
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	esExternal := []cerebrocrd.ElasticsearchExternalRef{
		{
			Name:    "test1",
			Address: "https://test1.domain.local",
		},
		{
			Name:    "test2",
			Address: "https://test2.domain.local",
		},
	}

	configMaps, err = buildConfigMaps(o, nil, esExternal)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_elasticsearch_external_targets.yml", configMaps[0], scheme.Scheme)
}
