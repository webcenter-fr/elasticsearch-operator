package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildConfigMap(t *testing.T) {

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
			Config: map[string]string{
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
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: ptr.To[bool](true),
			},
		},
	}

	configMap, err := BuildConfigMap(o, es)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_default.yml", configMap, test.CleanApi)

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
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](false),
			},
			Config: map[string]string{
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
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	configMap, err = BuildConfigMap(o, es)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_tls_disabled.yml", configMap, test.CleanApi)

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
			Config: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"fake"},
				},
				ElasticsearchCaSecretRef: &v1.LocalObjectReference{
					Name: "custom-ca-es",
				},
			},
		},
	}

	configMap, err = BuildConfigMap(o, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_external_es_custom_ca_es.yml", configMap, test.CleanApi)

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
			Config: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
				ElasticsearchCaSecretRef: &v1.LocalObjectReference{
					Name: "custom-ca-es",
				},
			},
		},
	}

	configMap, err = BuildConfigMap(o, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_managed_es_custom_ca_es.yml", configMap, test.CleanApi)

}
