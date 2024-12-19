package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
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

	configMaps, err = buildConfigMaps(o)
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

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_config.yml", &configMaps[0], scheme.Scheme)

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

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_pipeline.yml", &configMaps[0], scheme.Scheme)

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

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_pattern.yml", &configMaps[0], scheme.Scheme)

	// When exporter Prometheus
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
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: ptr.To[bool](true),
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_exporter.yml", &configMaps[0], scheme.Scheme)
}
