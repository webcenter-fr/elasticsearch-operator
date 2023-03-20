package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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
			Tls: ElasticsearchTlsSpec{
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
			Tls: ElasticsearchTlsSpec{
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
	o.Spec.Endpoint = ElasticsearchEndpointSpec{
		Ingress: &ElasticsearchIngressSpec{
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
	o.Spec.Endpoint = ElasticsearchEndpointSpec{
		LoadBalancer: &ElasticsearchLoadBalancerSpec{
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
			Tls: ElasticsearchTlsSpec{
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
			Tls: ElasticsearchTlsSpec{
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
			Monitoring: ElasticsearchMonitoringSpec{
				Prometheus: &ElasticsearchPrometheusSpec{
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
			Monitoring: ElasticsearchMonitoringSpec{
				Prometheus: &ElasticsearchPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsMetricbeatMonitoring(t *testing.T) {
	var o *Elasticsearch

	// With default values
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Monitoring: ElasticsearchMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring())

	// When disabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Monitoring: ElasticsearchMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())
}

func TestIsPersistence(t *testing.T) {
	var o *ElasticsearchNodeGroupSpec

	// With default value
	o = &ElasticsearchNodeGroupSpec{}
	assert.False(t, o.IsPersistence())

	// When persistence is not enabled
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{},
	}

	assert.False(t, o.IsPersistence())

	// When claim PVC is set
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{
			VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
		},
	}

	assert.True(t, o.IsPersistence())

	// When volume is set
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{
			Volume: &v1.VolumeSource{},
		},
	}

	assert.True(t, o.IsPersistence())

}
