package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServices(t *testing.T) {
	var (
		err      error
		services []corev1.Service
		o        *logstashcrd.Logstash
	)

	pathType := networkingv1.PathTypePrefix

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.NotEmpty(t, services)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_default.yaml", &services[0], scheme.Scheme)

	// When service is specified
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Services: []shared.Service{
				{
					Name: "my-service",
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"anno1": "value1",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:     "my-port",
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
						Selector: map[string]string{
							"dd": "toto",
						},
					},
				},
			},
		},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(services))
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_custom.yaml", &services[0], scheme.Scheme)

	// When ingress is specified
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Ingresses: []shared.Ingress{
				{
					Name: "my-ingress",
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"anno1": "value1",
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.cluster.local",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/",
												PathType: &pathType,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "my-service",
														Port: networkingv1.ServiceBackendPort{Number: 8081},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					ContainerPortProtocol: v1.ProtocolTCP,
					ContainerPort:         8080,
				},
			},
		},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(services))
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_ingress.yaml", &services[0], scheme.Scheme)

	// When route is specified
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Routes: []shared.Route{
				{
					Name: "my-route",
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"anno1": "value1",
					},
					Spec: routev1.RouteSpec{
						Host: "test.cluster.local",
						Path: "/",
						To: routev1.RouteTargetReference{
							Kind: "Service",
							Name: "my-service",
						},
						Port: &routev1.RoutePort{
							TargetPort: intstr.FromInt(8081),
						},
					},
					ContainerPortProtocol: v1.ProtocolTCP,
					ContainerPort:         8080,
				},
			},
		},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(services))
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_route.yaml", &services[0], scheme.Scheme)
}
