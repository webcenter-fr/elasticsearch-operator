package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *elasticsearchapi.Elasticsearch
		i   *networkingv1.Ingress
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is disabled
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				Ingress: &elasticsearchapi.IngressSpec{
					Enabled: false,
				},
			},
		},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is enabled ans specify target service
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				Ingress: &elasticsearchapi.IngressSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
					Host:                "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				Ingress: &elasticsearchapi.IngressSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
	o = &elasticsearchapi.Elasticsearch{
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
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				Ingress: &elasticsearchapi.IngressSpec{
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
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Endpoint: elasticsearchapi.EndpointSpec{
				Ingress: &elasticsearchapi.IngressSpec{
					Enabled:             true,
					TargetNodeGroupName: "master",
					Host:                "my-test.cluster.local",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
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
