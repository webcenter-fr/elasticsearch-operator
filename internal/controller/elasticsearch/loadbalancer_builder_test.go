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
)

func TestBuildLoadbalancer(t *testing.T) {
	var (
		err      error
		services []corev1.Service
		o        *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
						Enabled: false,
					},
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
						Enabled: true,
					},
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/loadbalancer_with_target.yaml", &services[0], scheme.Scheme)

	// When load balancer is enabled without target node group
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
						Enabled: true,
					},
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/loadbalancer_without_target.yaml", &services[0], scheme.Scheme)

	// When load balancer is enabled with target node group that not exist
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
					EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
						Enabled: true,
					},
					TargetNodeGroupName: "master",
				},
			},
		},
	}

	_, err = buildLoadbalancers(o)
	assert.Error(t, err)
}
