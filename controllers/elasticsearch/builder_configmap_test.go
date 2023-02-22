package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildConfigMaps(t *testing.T) {

	var o *elasticsearchcrd.Elasticsearch

	// Normal
	o = &elasticsearchcrd.Elasticsearch{
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
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				Config: map[string]string{
					"elasticsearch.yml": `node.value: test
node.value2: test`,
					"log4j.yml": "log.test: test\n",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Config: map[string]string{
						"elasticsearch.yml": `
key.value: fake
node:
  value2: test2
  name: test
  roles:
    - 'master'`,
					},
				},
			},
		},
	}

	configMaps, err := BuildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_default.yml", &configMaps[0], test.CleanApi)

	// When TLS API is disabled
	o = &elasticsearchcrd.Elasticsearch{
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
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(false),
			},
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				Config: map[string]string{
					"elasticsearch.yml": `node.value: test
node.value2: test`,
					"log4j.yml": "log.test: test\n",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Config: map[string]string{
						"elasticsearch.yml": `
key.value: fake
node:
  value2: test2
  name: test
  roles:
    - 'master'`,
					},
				},
			},
		},
	}

	configMaps, err = BuildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_api_tls_disabled.yml", &configMaps[0], test.CleanApi)
}
