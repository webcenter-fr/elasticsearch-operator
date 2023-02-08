package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMaps(t *testing.T) {

	var (
		o          *beatcrd.Filebeat
		es         *elasticsearchcrd.Elasticsearch
		configMaps []corev1.ConfigMap
		err        error
	)

	// When default value
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	configMaps, err = BuildConfigMaps(o, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_default.yml", &configMaps[0], test.CleanApi)

	// When default value and logstasgh output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			LogstashRef: beatcrd.LogstashRef{
				ManagedLogstashRef: &beatcrd.LogstashManagedRef{
					Name: "test",
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "ls-ca",
				},
			},
		},
	}

	configMaps, err = BuildConfigMaps(o, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_default_logstash.yml", &configMaps[0], test.CleanApi)

	// When default value and elasticsearch output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	configMaps, err = BuildConfigMaps(o, es)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_default_elasticsearch.yml", &configMaps[0], test.CleanApi)

	// When config
	o = &beatcrd.Filebeat{
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
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Config: map[string]string{
				"filebeat.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	configMaps, err = BuildConfigMaps(o, es)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_config.yml", &configMaps[0], test.CleanApi)

	// When module
	o = &beatcrd.Filebeat{
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
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Module: map[string]string{
				"module.conf": "foo = bar",
			},
		},
	}

	configMaps, err = BuildConfigMaps(o, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_module.yml", &configMaps[1], test.CleanApi)

}