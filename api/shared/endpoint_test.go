package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIngressEnabled(t *testing.T) {
	// With default values
	o := EndpointSpec{}
	assert.False(t, o.IsIngressEnabled())

	// When Ingress is specified but disabled
	o = EndpointSpec{
		Ingress: &EndpointIngressSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsIngressEnabled())

	// When ingress is enabled
	o.Ingress.Enabled = true
	assert.True(t, o.IsIngressEnabled())
}

func TestIsRouteEnabled(t *testing.T) {
	// With default values
	o := EndpointSpec{}
	assert.False(t, o.IsRouteEnabled())

	// When Ingress is specified but disabled
	o = EndpointSpec{
		Route: &EndpointRouteSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsRouteEnabled())

	// When ingress is enabled
	o.Route.Enabled = true
	assert.True(t, o.IsRouteEnabled())
}

func TestIsLoadBalancerEnabled(t *testing.T) {
	// With default values
	o := EndpointSpec{}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified but disabled
	o = EndpointSpec{
		LoadBalancer: &EndpointLoadBalancerSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}
