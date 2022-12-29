package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildIngress(t *testing.T) {
	var (
		err error
		o   *kibanaapi.Kibana
		i   *networkingv1.Ingress
	)

	// With default values
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is disabled
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{
			Endpoint: kibanaapi.EndpointSpec{
				Ingress: &kibanaapi.IngressSpec{
					Enabled: false,
				},
			},
		},
	}
	i, err = BuildIngress(o)
	assert.NoError(t, err)
	assert.Nil(t, i)

	// When ingress is enabled
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{
			Endpoint: kibanaapi.EndpointSpec{
				Ingress: &kibanaapi.IngressSpec{
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
	o = &kibanaapi.Kibana{
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
		Spec: kibanaapi.KibanaSpec{
			Endpoint: kibanaapi.EndpointSpec{
				Ingress: &kibanaapi.IngressSpec{
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
