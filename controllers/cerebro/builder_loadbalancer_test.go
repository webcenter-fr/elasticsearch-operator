package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLoadbalancer(t *testing.T) {

	var (
		err     error
		service *corev1.Service
		o       *cerebrocrd.Cerebro
	)

	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is disabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: cerebrocrd.CerebroEndpointSpec{
				LoadBalancer: &cerebrocrd.CerebroLoadBalancerSpec{
					Enabled: false,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	assert.Nil(t, service)

	// When load balancer is enabled
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Endpoint: cerebrocrd.CerebroEndpointSpec{
				LoadBalancer: &cerebrocrd.CerebroLoadBalancerSpec{
					Enabled: true,
				},
			},
		},
	}

	service, err = BuildLoadbalancer(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/loadbalancer_without_target.yaml", service, test.CleanApi)

}
