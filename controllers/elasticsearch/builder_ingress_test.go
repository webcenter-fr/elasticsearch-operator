package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		i   *networkingv1.Ingress
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.EndpointSpec{
				Ingress: &elasticsearchcrd.IngressSpec{
					Enabled: false,
				},
			},
		},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is enabled ans specify target service
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.EndpointSpec{
				Ingress: &elasticsearchcrd.IngressSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
					Host:                "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchcrd.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
				{
					Name:     "data",
					Replicas: 1,
				},
			},
		},
	}
	i, err = BuildIngress(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/ingress_with_target.yml", i, test.CleanApi)

	// When ingress is enabled without specify TargetNodeGroupName
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.EndpointSpec{
				Ingress: &elasticsearchcrd.IngressSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchcrd.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
				{
					Name:     "data",
					Replicas: 1,
				},
			},
		},
	}
	i, err = BuildIngress(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/ingress_without_target.yml", i, test.CleanApi)

	// When ingress is enabled and specify all options
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
			Labels: map[string]string{
				"globalLabel": "globalLabel",
			},
			Annotations: map[string]string{
				"globalAnnotation": "globalAnnotation",
			},
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.EndpointSpec{
				Ingress: &elasticsearchcrd.IngressSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
					Host:                "my-test.cluster.local",
					SecretRef: &v1.LocalObjectReference{
						Name: "my-secret",
					},
					Labels: map[string]string{
						"ingressLabel": "ingressLabel",
					},
					Annotations: map[string]string{
						"annotationLabel": "annotationLabel",
					},
					IngressSpec: &networkingv1.IngressSpec{
						IngressClassName: pointer.String("toto"),
					},
				},
			},
			NodeGroups: []elasticsearchcrd.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
				{
					Name:     "data",
					Replicas: 1,
				},
			},
		},
	}
	i, err = BuildIngress(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/ingress_with_all_options.yml", i, test.CleanApi)

	// When target nodeGroup not exist
	// When ingress is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.EndpointSpec{
				Ingress: &elasticsearchcrd.IngressSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
					Host:                "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchcrd.NodeGroupSpec{
				{
					Name:     "data",
					Replicas: 1,
				},
			},
		},
	}
	_, err = BuildIngress(o)
	assert.Error(t, err)
}
