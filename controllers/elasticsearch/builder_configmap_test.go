package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildConfigMaps(t *testing.T) {

	var o *elasticsearchapi.Elasticsearch

	// Normal
	o = &elasticsearchapi.Elasticsearch{
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
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				Config: map[string]string{
					"elasticsearch.yml": `node.value: test
node.value2: test`,
					"log4j.yml": "log.test: test\n",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
	o = &elasticsearchapi.Elasticsearch{
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
		Spec: elasticsearchapi.ElasticsearchSpec{
			Tls: elasticsearchapi.TlsSpec{
				Enabled: pointer.Bool(false),
			},
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				Config: map[string]string{
					"elasticsearch.yml": `node.value: test
node.value2: test`,
					"log4j.yml": "log.test: test\n",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
