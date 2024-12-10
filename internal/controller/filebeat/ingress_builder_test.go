package filebeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildIngresses(t *testing.T) {
	var (
		err       error
		o         *beatcrd.Filebeat
		ingresses []networkingv1.Ingress
	)

	pathType := networkingv1.PathTypePrefix

	// With default values
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}
	ingresses, err = buildIngresses(o)
	assert.NoError(t, err)
	assert.Empty(t, ingresses)

	// When ingress is defined
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
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
	ingresses, err = buildIngresses(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(ingresses))
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_default.yml", &ingresses[0], scheme.Scheme)

}
