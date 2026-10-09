package filebeat

import (
	"fmt"
	"testing"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildConfigMaps(t *testing.T) {
	var (
		o                  *beatcrd.Filebeat
		es                 *elasticsearchcrd.Elasticsearch
		configMaps         []*corev1.ConfigMap
		err                error
		s                  *corev1.Secret
		ls                 *logstashcrd.Logstash
		discoverEnvSecret  *corev1.Secret
		discoverFileSecret *corev1.Secret
	)

	// When default value
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	configMaps, err = buildConfigMaps(o, nil, nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default.yml", configMaps[0], scheme.Scheme)

	// When default value and logstasgh output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			LogstashRef: &beatcrd.FilebeatLogstashRef{
				ManagedLogstashRef: &beatcrd.FilebeatLogstashManagedRef{
					Name:          "test",
					TargetService: "beat",
					Port:          5003,
				},
			},
		},
	}

	ls = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}

	configMaps, err = buildConfigMaps(o, nil, ls, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_logstash.yml", configMaps[0], scheme.Scheme)

	// When logstasgh output that managed pki with beat certificates
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			LogstashRef: &beatcrd.FilebeatLogstashRef{
				ManagedLogstashRef: &beatcrd.FilebeatLogstashManagedRef{
					Name:          "test",
					TargetService: "beat",
					Port:          5003,
				},
			},
		},
	}

	ls = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Pki: logstashcrd.LogstashPkiSpec{
				Tls: map[string]logstashcrd.LogstashTlsSpec{
					"filebeat.crt": {
						Consumer: "beat",
					},
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o, nil, ls, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_pki_logstash.yml", configMaps[0], scheme.Scheme)

	// When default value and logstasgh output and logstashCaSecretRef
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			LogstashRef: &beatcrd.FilebeatLogstashRef{
				ManagedLogstashRef: &beatcrd.FilebeatLogstashManagedRef{
					Name:          "test",
					TargetService: "beat",
					Port:          5003,
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "ls-ca",
				},
			},
		},
	}

	ls = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "logstash-ca",
		},
		Data: map[string][]byte{
			"filebeat.crt": {},
		},
	}

	configMaps, err = buildConfigMaps(o, nil, ls, nil, s, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_logstash_with_ca_secret.yml", configMaps[0], scheme.Scheme)

	// When default value and elasticsearch output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: &shared.ElasticsearchRef{
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

	configMaps, err = buildConfigMaps(o, es, nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_elasticsearch.yml", configMaps[0], scheme.Scheme)

	// When default value and elasticsearch output and elasticsearchCaSecretRef
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: &shared.ElasticsearchRef{
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

	configMaps, err = buildConfigMaps(o, nil, nil, s, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default_elasticsearch_with_ca_secret.yml", configMaps[0], scheme.Scheme)

	// When discover Kafka output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			DiscoverRef: []*discovercrd.DiscoverRef{
				{
					Kafka: &corev1.LocalObjectReference{
						Name: "test-kafka",
					},
				},
			},
			DiscoverOutputName: ptr.To("test-kafka"),
		},
	}
	discoverEnvSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-kafka-env",
		},
		Data: map[string][]byte{
			"KAFKA_BOOTSTRAP_SERVERS_TEST_KAFKA": []byte("kafka1:9092,kafka2:9092"),
		},
	}
	discoverFileSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-kafka-file",
			Annotations: map[string]string{
				fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-kafka",
			},
		},
		Data: map[string][]byte{
			"ca.crt":   []byte("ca"),
			"user.crt": []byte("user-crt"),
			"user.key": []byte("user-key"),
		},
	}

	configMaps, err = buildConfigMaps(o, nil, nil, nil, nil, discovercrd.DiscoverTypeKafka, discoverEnvSecret, discoverFileSecret)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_discover_kafka.yml", configMaps[0], scheme.Scheme)

	// When discover Logstash output
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			DiscoverRef: []*discovercrd.DiscoverRef{
				{
					Logstash: &corev1.LocalObjectReference{
						Name: "test-logstash",
					},
				},
			},
			DiscoverOutputName: ptr.To("test-logstash"),
		},
	}
	discoverEnvSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-logstash-env",
		},
		Data: map[string][]byte{
			"LOGSTASH_HOSTS_TEST_LOGSTASH": []byte("logstash1:5000,logstash2:5000"),
		},
	}
	discoverFileSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-logstash-file",
			Annotations: map[string]string{
				fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-logstash",
			},
		},
		Data: map[string][]byte{
			"ca.crt":   []byte("ca"),
			"user.crt": []byte("user-crt"),
			"user.key": []byte("user-key"),
		},
	}

	configMaps, err = buildConfigMaps(o, nil, nil, nil, nil, discovercrd.DiscoverTypeLogstash, discoverEnvSecret, discoverFileSecret)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_discover_logstash.yml", configMaps[0], scheme.Scheme)

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
			ElasticsearchRef: &shared.ElasticsearchRef{
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

	configMaps, err = buildConfigMaps(o, es, nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_config.yml", configMaps[0], scheme.Scheme)

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
			ElasticsearchRef: &shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Modules: &apis.MapAny{
				Data: map[string]any{
					"module.yaml": []map[string]any{
						{
							"foo": "bar",
						},
					},
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o, nil, nil, nil, nil, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configMaps))
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_module.yml", configMaps[1], scheme.Scheme)
}
