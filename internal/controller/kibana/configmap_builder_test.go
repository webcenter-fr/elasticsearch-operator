package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildConfigMaps(t *testing.T) {
	var (
		o  *kibanacrd.Kibana
		es *elasticsearchcrd.Elasticsearch
	)

	// Normal
	o = &kibanacrd.Kibana{
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
		Spec: kibanacrd.KibanaSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Config: &apis.MapAny{
				Data: map[string]any{
					"node.test": "test",
				},
			},
			ExtraConfigs: map[string]string{
				"kibana.yml": `node.value: test
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
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](true),
			},
		},
	}

	configMaps, err := buildConfigMaps(o, es)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default.yml", configMaps[0], scheme.Scheme)

	// When TLS is disabled
	o = &kibanacrd.Kibana{
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
		Spec: kibanacrd.KibanaSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
			},
			ExtraConfigs: map[string]string{
				"kibana.yml": `node.value: test
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
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	configMaps, err = buildConfigMaps(o, es)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_tls_disabled.yml", configMaps[0], scheme.Scheme)

	// When external elasticsearch with custom CA elasticsearch
	o = &kibanacrd.Kibana{
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
		Spec: kibanacrd.KibanaSpec{
			ExtraConfigs: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"fake"},
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-es",
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_external_es_custom_ca_es.yml", configMaps[0], scheme.Scheme)

	// When managed elasticsearch with custom CA elasticsearch
	o = &kibanacrd.Kibana{
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
		Spec: kibanacrd.KibanaSpec{
			ExtraConfigs: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-es",
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_managed_es_custom_ca_es.yml", configMaps[0], scheme.Scheme)
}
