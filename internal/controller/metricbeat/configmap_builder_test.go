package metricbeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildConfigMaps(t *testing.T) {
	var (
		o          *beatcrd.Metricbeat
		es         *elasticsearchcrd.Elasticsearch
		configMaps []corev1.ConfigMap
		err        error
		s          *corev1.Secret
	)

	// When default value
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	configMaps, err = buildConfigMaps(o, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default.yml", &configMaps[0], scheme.Scheme)

	// When default value with managed elasticsearch
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{
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

	configMaps, err = buildConfigMaps(o, es, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_elasticsearch.yml", &configMaps[0], scheme.Scheme)

	// When default value and elasticsearch output and elasticsearchCaSecretRef
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://external-es"},
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "es-ca",
				},
			},
		},
	}
	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "elasticsearch-ca",
		},
		Data: map[string][]byte{
			"elasticsearch.crt": {},
		},
	}

	configMaps, err = buildConfigMaps(o, nil, s)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_elasticsearch_with_ca_secret.yml", &configMaps[0], scheme.Scheme)

	// When config
	o = &beatcrd.Metricbeat{
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
		Spec: beatcrd.MetricbeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Config: &apis.MapAny{
				Data: map[string]any{
					"node.foo": "bar",
				},
			},
			ExtraConfigs: map[string]string{
				"metricbeat.yml": `node.value: test
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

	configMaps, err = buildConfigMaps(o, es, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_config.yml", &configMaps[0], scheme.Scheme)

	// When module
	o = &beatcrd.Metricbeat{
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
		Spec: beatcrd.MetricbeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Modules: map[string][]apis.MapAny{
				"module.yaml": []apis.MapAny{
					{
						Data: map[string]any{
							"foo": "bar",
						},
					},
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

	configMaps, err = buildConfigMaps(o, es, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_module.yml", &configMaps[1], scheme.Scheme)
}
