package cerebro

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildLoadbalancer(t *testing.T) {

	var (
		err      error
		services []corev1.Service
		o        *cerebrocrd.Cerebro
	)

	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is disabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: shared.EndpointSpec{
				LoadBalancer: &shared.EndpointLoadBalancerSpec{
					Enabled: false,
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When load balancer is enabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: shared.EndpointSpec{
				LoadBalancer: &shared.EndpointLoadBalancerSpec{
					Enabled: true,
				},
			},
		},
	}

	services, err = buildLoadbalancers(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/loadbalancer_without_target.yaml", &services[0], scheme.Scheme)

}
