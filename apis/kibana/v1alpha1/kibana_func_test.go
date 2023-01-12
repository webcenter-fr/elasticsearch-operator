package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestIsSelfManagedSecretForTls(t *testing.T) {
	var o *Kibana

	// With default settings
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.True(t, o.IsSelfManagedSecretForTls())

	// When TLS is enabled but without specify secrets
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
			},
		},
	}
	assert.True(t, o.IsSelfManagedSecretForTls())

	// When TLS is enabled and pecify secrets
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "my-secret",
				},
			},
		},
	}
	assert.False(t, o.IsSelfManagedSecretForTls())

}

func TestIsTlsEnabled(t *testing.T) {
	var o *Kibana

	// With default values
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.True(t, o.IsTlsEnabled())

	// When enabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
			},
		},
	}
	assert.True(t, o.IsTlsEnabled())

	// When disabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(false),
			},
		},
	}
	assert.False(t, o.IsTlsEnabled())
}

func TestIsIngressEnabled(t *testing.T) {

	// With default values
	o := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsIngressEnabled())

	// When Ingress is specified but disabled
	o.Spec.Endpoint = EndpointSpec{
		Ingress: &IngressSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsIngressEnabled())

	// When ingress is enabled
	o.Spec.Endpoint.Ingress.Enabled = true

}

func TestIsLoadBalancerEnabled(t *testing.T) {
	// With default values
	o := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified but disabled
	o.Spec.Endpoint = EndpointSpec{
		LoadBalancer: &LoadBalancerSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.Spec.Endpoint.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}
