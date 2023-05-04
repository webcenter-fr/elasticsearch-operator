package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *cerebrocrd.Cerebro
		i   *networkingv1.Ingress
	)

	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is disabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: cerebrocrd.CerebroEndpointSpec{
				Ingress: &cerebrocrd.CerebroIngressSpec{
					Enabled: false,
				},
			},
		},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is enabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: cerebrocrd.CerebroEndpointSpec{
				Ingress: &cerebrocrd.CerebroIngressSpec{
					Enabled: true,
					Host:    "my-test.cluster.local",
				},
			},
		},
	}
	i, err = BuildIngress(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/ingress_without_target.yml", i, test.CleanApi)

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
			Endpoint: cerebrocrd.CerebroEndpointSpec{
				Ingress: &cerebrocrd.CerebroIngressSpec{
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
						IngressClassName: pointer.String("toto"),
					},
				},
			},
		},
	}
	i, err = BuildIngress(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/ingress_with_all_options.yml", i, test.CleanApi)
}
