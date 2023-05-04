package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLoadbalancer(t *testing.T) {

	var (
		err     error
		service *corev1.Service
		o       *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					Enabled: false,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
				{
					Name:     "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/loadbalancer_with_target.yaml", service, test.CleanApi)

	// When load balancer is enabled without target node group
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
				{
					Name:     "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					Enabled: true,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/loadbalancer_without_target.yaml", service, test.CleanApi)

	// When load balancer is enabled with target node group that not exist
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	_, err = BuildLoadbalancer(o)
	assert.Error(t, err)
}
