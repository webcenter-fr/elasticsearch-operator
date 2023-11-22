package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetStatus(t *testing.T) {
	status := KibanaStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

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
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](true),
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
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](true),
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
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](true),
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
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
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
	o.Spec.Endpoint = shared.EndpointSpec{
		Ingress: &shared.EndpointIngressSpec{
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
	o := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified but disabled
	o.Spec.Endpoint = shared.EndpointSpec{
		LoadBalancer: &shared.EndpointLoadBalancerSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.Spec.Endpoint.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}

func TestIsPdb(t *testing.T) {
	var o Kibana

	// When default
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsPdb())

	// When default with replica > 1
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			Deployment: KibanaDeploymentSpec{
				Replicas: 2,
			},
		},
	}
	assert.True(t, o.IsPdb())

	// When PDB is set
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			Deployment: KibanaDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb())

}
