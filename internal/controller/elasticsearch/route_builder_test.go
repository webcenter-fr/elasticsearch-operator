package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildRoute(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		r   []routev1.Route
	)

	sch := scheme.Scheme
	if err := routev1.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	r, err = buildRoutes(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, r)

	// When route is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
						Enabled: false,
					},
				},
			},
		},
	}
	r, err = buildRoutes(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, r)

	// When route is enabled and specify target service
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
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
	r, err = buildRoutes(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_with_target.yml", &r[0], sch)

	// When route is enabled without specify TargetNodeGroupName
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
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
	r, err = buildRoutes(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_without_target.yml", &r[0], sch)

	// When route is enabled and specify all options
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
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
						SecretRef: &v1.LocalObjectReference{
							Name: "my-secret",
						},
						TlsEnabled: ptr.To(true),
						Labels: map[string]string{
							"ingressLabel": "ingressLabel",
						},
						Annotations: map[string]string{
							"annotationLabel": "annotationLabel",
						},
						RouteSpec: &routev1.RouteSpec{
							Path: "/fake",
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
	r, err = buildRoutes(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_with_all_options.yml", &r[0], sch)

	// When target nodeGroup not exist
	// When route is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
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
	_, err = buildRoutes(o, nil)
	assert.Error(t, err)

	// When route is enabled and backend is over TLS
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
				Route: &elasticsearchcrd.ElasticsearchRouteSpec{
					EndpointRouteSpec: shared.EndpointRouteSpec{
						Enabled: true,
						Host:    "my-test.cluster.local",
					},
				},
			},
		},
	}
	secretTls := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("my fake certificate"),
		},
	}
	r, err = buildRoutes(o, secretTls)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_with_tls_backend.yml", &r[0], sch)
}
