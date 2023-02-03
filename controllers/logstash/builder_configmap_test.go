package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMaps(t *testing.T) {

	var (
		o          *logstashcrd.Logstash
		configMaps []corev1.ConfigMap
		err        error
	)

	// When default value
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	configMaps, err = BuildConfigMaps(o)
	assert.NoError(t, err)
	assert.Empty(t, configMaps)

	// When config
	o = &logstashcrd.Logstash{
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
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Config: map[string]string{
				"logstash.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
		},
	}

	configMaps, err = BuildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_config.yml", &configMaps[0], test.CleanApi)

	// When pipeline
	o = &logstashcrd.Logstash{
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
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Pipeline: map[string]string{
				"pipeline.conf": "foo = bar",
			},
		},
	}

	configMaps, err = BuildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_pipeline.yml", &configMaps[0], test.CleanApi)

	// When pattern
	o = &logstashcrd.Logstash{
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
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Pattern: map[string]string{
				"pattern.conf": "foo = bar",
			},
		},
	}

	configMaps, err = BuildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile(t, "testdata/configmap_pattern.yml", &configMaps[0], test.CleanApi)

}
