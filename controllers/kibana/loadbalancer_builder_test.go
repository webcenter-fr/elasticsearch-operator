package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildLoadbalancer(t *testing.T) {

	var (
		err      error
		services []corev1.Service
		o        *kibanacrd.Kibana
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Endpoint: kibanacrd.KibanaEndpointSpec{
				LoadBalancer: &kibanacrd.KibanaLoadBalancerSpec{
					Enabled: false,
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is enabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Endpoint: kibanacrd.KibanaEndpointSpec{
				LoadBalancer: &kibanacrd.KibanaLoadBalancerSpec{
					Enabled: true,
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/loadbalancer_without_target.yaml", &services[0], scheme.Scheme)

}
