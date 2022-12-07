package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLoadbalancer(t *testing.T) {

	var (
		err error
		service *corev1.Service
		o *elasticsearchapi.Elasticsearch
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is disabled
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				LoadBalancer: &elasticsearchapi.LoadBalancerSpec{
					Enabled: false,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is enabled
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 3,
				},
				{
					Name: "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchapi.EndpointSpec{
				LoadBalancer: &elasticsearchapi.LoadBalancerSpec{
					Enabled: true,
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/loadbalancer_with_target.yaml", service, test.CleanApi)

	// When load balancer is enabled without target node group
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 3,
				},
				{
					Name: "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchapi.EndpointSpec{
				LoadBalancer: &elasticsearchapi.LoadBalancerSpec{
					Enabled: true,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/loadbalancer_without_target.yaml", service, test.CleanApi)

	// When load balancer is enabled with target node group that not exist
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "data",
					Replicas: 1,
				},
			},
			Endpoint: elasticsearchapi.EndpointSpec{
				LoadBalancer: &elasticsearchapi.LoadBalancerSpec{
					Enabled: true,
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	_, err = BuildLoadbalancer(o)
	assert.Error(t, err)
}