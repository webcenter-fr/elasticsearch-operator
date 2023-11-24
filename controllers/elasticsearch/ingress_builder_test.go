package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		i   []networkingv1.Ingress
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	i, err = buildIngresses(o)
	assert.NoError(t, err)
	assert.Empty(t, i)

	// When ingress is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
					EndpointIngressSpec: shared.EndpointIngressSpec{
						Enabled: false,
					},
				},
			},
		},
	}
	i, err = buildIngresses(o)
	assert.NoError(t, err)
	assert.Empty(t, i)

	// When ingress is enabled ans specify target service
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
					EndpointIngressSpec: shared.EndpointIngressSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
					},
					TargetNodeGroupName: "master",
				},
			},
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
		},
	}
	i, err = buildIngresses(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_with_target.yml", &i[0], scheme.Scheme)

	// When ingress is enabled without specify TargetNodeGroupName
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
					EndpointIngressSpec: shared.EndpointIngressSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
					},
				},
			},
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
		},
	}
	i, err = buildIngresses(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_without_target.yml", &i[0], scheme.Scheme)

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
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
					EndpointIngressSpec: shared.EndpointIngressSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
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
							IngressClassName: ptr.To[string]("toto"),
						},
					},
					TargetNodeGroupName: "master",
				},
			},
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
		},
	}
	i, err = buildIngresses(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_with_all_options.yml", &i[0], scheme.Scheme)

	// When target nodeGroup not exist
	// When ingress is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
					EndpointIngressSpec: shared.EndpointIngressSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
					},
					TargetNodeGroupName: "master",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}
	_, err = buildIngresses(o)
	assert.Error(t, err)
}
