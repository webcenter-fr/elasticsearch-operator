package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
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

	configMaps, err := buildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_default.yml", &configMaps[0], scheme.Scheme)

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
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
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

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_api_tls_disabled.yml", &configMaps[0], scheme.Scheme)

	// When cluster is not yet bootstrapped
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_not_bootstrapping.yml", &configMaps[1], scheme.Scheme)

	// When cluster is bootstrapped
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
		Status: elasticsearchcrd.ElasticsearchStatus{
			IsBootstrapping: ptr.To[bool](true),
		},
	}

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_bootstrapping.yml", &configMaps[1], scheme.Scheme)

	// When cluster is not yet bootstrapped and single node
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	configMaps, err = buildConfigMaps(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.ConfigMap](t, "testdata/configmap_not_bootstrapping_single.yml", &configMaps[1], scheme.Scheme)
}

func TestComputeInitialMasterNodes(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// With only one master
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-es-0, test-master-es-1, test-master-es-2", computeInitialMasterNodes(o))

	// With multiple node groups
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "all",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-all-es-0, test-all-es-1, test-all-es-2, test-master-es-0, test-master-es-1, test-master-es-2", computeInitialMasterNodes(o))
}

func TestComputeDiscoverySeedHosts(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// With only one master
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-headless-es", computeDiscoverySeedHosts(o))

	// With multiple node groups
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "all",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-all-headless-es, test-master-headless-es", computeDiscoverySeedHosts(o))
}
