package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreberoGetStatus(t *testing.T) {
	status := CerebroStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

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
	assert.True(t, o.IsIngressEnabled())

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
