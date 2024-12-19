package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
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
		o   *kibanacrd.Kibana
		r   []routev1.Route
	)

	sch := scheme.Scheme
	if err := routev1.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}
	r, err = buildRoutes(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, r)

	// When route is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Endpoint: shared.EndpointSpec{
				Route: &shared.EndpointRouteSpec{
					Enabled: false,
				},
			},
		},
	}
	r, err = buildRoutes(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, r)

	// When route is enabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Endpoint: shared.EndpointSpec{
				Route: &shared.EndpointRouteSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
				},
			},
		},
	}
	r, err = buildRoutes(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_without_target.yml", &r[0], sch)

	// When route is enabled and specify all options
	o = &kibanacrd.Kibana{
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
		Spec: kibanacrd.KibanaSpec{
			Endpoint: shared.EndpointSpec{
				Route: &shared.EndpointRouteSpec{
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
					RouteSpec: &routev1.RouteSpec{
						Path: "/fake",
					},
					TlsEnabled: ptr.To[bool](true),
				},
			},
		},
	}
	r, err = buildRoutes(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_with_all_options.yml", &r[0], sch)

	// When route is enabled and backend is over TLS
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Endpoint: shared.EndpointSpec{
				Route: &shared.EndpointRouteSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
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
