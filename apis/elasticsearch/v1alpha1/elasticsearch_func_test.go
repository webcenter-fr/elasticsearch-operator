package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestIsSelfManagedSecretForTlsApi(t *testing.T) {
	var o *Elasticsearch

	// With default settings
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.True(t, o.IsSelfManagedSecretForTlsApi())

	// When TLS is enabled but without specify secrets
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
			},
		},
	}
	assert.True(t, o.IsSelfManagedSecretForTlsApi())

	// When TLS is enabled and pecify secrets
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "my-secret",
				},
			},
		},
	}
	assert.False(t, o.IsSelfManagedSecretForTlsApi())

}

func TestIsIngressEnabled(t *testing.T) {

	// With default values
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
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
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
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

func TestIsTlsApiEnabled(t *testing.T) {
	var o *Elasticsearch

	// With default values
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.True(t, o.IsTlsApiEnabled())

	// When enabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(true),
			},
		},
	}
	assert.True(t, o.IsTlsApiEnabled())

	// When disabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Tls: TlsSpec{
				Enabled: pointer.Bool(false),
			},
		},
	}
	assert.False(t, o.IsTlsApiEnabled())
}

func TestIsSetVMMaxMapCount(t *testing.T) {
	var o *Elasticsearch

	// With default values
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.True(t, o.IsSetVMMaxMapCount())

	// When enabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			SetVMMaxMapCount: pointer.Bool(true),
		},
	}
	assert.True(t, o.IsSetVMMaxMapCount())

	// When disabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			SetVMMaxMapCount: pointer.Bool(false),
		},
	}
	assert.False(t, o.IsSetVMMaxMapCount())
}

func TestIsPrometheusMonitoring(t *testing.T) {
	var o *Elasticsearch

	// With default values
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}
