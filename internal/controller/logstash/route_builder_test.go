package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildRoutes(t *testing.T) {
	var (
		err    error
		o      *logstashcrd.Logstash
		routes []*routev1.Route
	)

	sch := scheme.Scheme
	if err := routev1.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}
	routes, err = buildRoutes(o)
	assert.NoError(t, err)
	assert.Empty(t, routes)

	// When route is defined
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Routes: []shared.Route{
				{
					Name: "my-ingress",
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"anno1": "value1",
					},
					Spec: routev1.RouteSpec{
						Host: "test.cluster.local",
						To: routev1.RouteTargetReference{
							Kind: "Service",
							Name: "my-service",
						},
						Port: &routev1.RoutePort{
							TargetPort: intstr.FromInt(8081),
						},
						Path: "/",
					},
					ContainerPortProtocol: v1.ProtocolTCP,
					ContainerPort:         8080,
				},
			},
		},
	}
	routes, err = buildRoutes(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(routes))
	test.EqualFromYamlFile[*routev1.Route](t, "testdata/route_default.yml", routes[0], sch)
}
