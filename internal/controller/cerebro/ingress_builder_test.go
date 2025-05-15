package cerebro

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *cerebrocrd.Cerebro
		i   []*networkingv1.Ingress
	)

	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}
	i, err = buildIngresses(o)
	assert.NoError(t, err)
	assert.Empty(t, i)

	// When ingress is disabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: shared.EndpointSpec{
				Ingress: &shared.EndpointIngressSpec{
					Enabled: false,
				},
			},
		},
	}
	i, err = buildIngresses(o)
	assert.NoError(t, err)
	assert.Empty(t, i)

	// When ingress is enabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: shared.EndpointSpec{
				Ingress: &shared.EndpointIngressSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
				},
			},
		},
	}
	i, err = buildIngresses(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_without_target.yml", i[0], scheme.Scheme)

	// When ingress is enabled and specify all options
	o = &cerebrocrd.Cerebro{
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
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: shared.EndpointSpec{
				Ingress: &shared.EndpointIngressSpec{
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
			},
		},
	}
	i, err = buildIngresses(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.Ingress](t, "testdata/ingress_with_all_options.yml", i[0], scheme.Scheme)
}
