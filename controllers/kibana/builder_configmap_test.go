package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
			ElasticsearchRef: &kibanacrd.ElasticsearchRef{
				Name: "test",
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
			Tls: elasticsearchcrd.TlsSpec{
				Enabled: pointer.Bool(true),
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
			ElasticsearchRef: &kibanacrd.ElasticsearchRef{
				Name: "test",
			},
			Tls: kibanacrd.TlsSpec{
				Enabled: pointer.Bool(false),
			},
			Config: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
		},
	}

	configMap, err = BuildConfigMap(o, es)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_tls_disabled.yml", configMap, test.CleanApi)
}
