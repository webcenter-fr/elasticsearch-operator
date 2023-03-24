package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsIngressEnabled(t *testing.T) {

	// With default values
	o := &Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: CerebroSpec{},
	}
	assert.False(t, o.IsIngressEnabled())

	// When Ingress is specified but disabled
	o.Spec.Endpoint = CerebroEndpointSpec{
		Ingress: &CerebroIngressSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsIngressEnabled())

	// When ingress is enabled
	o.Spec.Endpoint.Ingress.Enabled = true

}

func TestIsLoadBalancerEnabled(t *testing.T) {
	// With default values
	o := &Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: CerebroSpec{},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified but disabled
	o.Spec.Endpoint = CerebroEndpointSpec{
		LoadBalancer: &CerebroLoadBalancerSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.Spec.Endpoint.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}
