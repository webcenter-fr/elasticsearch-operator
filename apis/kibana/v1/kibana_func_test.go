package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
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
			Tls: KibanaTlsSpec{
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
			Tls: KibanaTlsSpec{
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
			Tls: KibanaTlsSpec{
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
			Tls: KibanaTlsSpec{
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
	o.Spec.Endpoint = KibanaEndpointSpec{
		Ingress: &KibanaIngressSpec{
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
	o.Spec.Endpoint = KibanaEndpointSpec{
		LoadBalancer: &KibanaLoadBalancerSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.Spec.Endpoint.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}

func TestIsPrometheusMonitoring(t *testing.T) {
	var o *Kibana

	// With default values
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Monitoring: KibanaMonitoringSpec{
				Prometheus: &KibanaPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Monitoring: KibanaMonitoringSpec{
				Prometheus: &KibanaPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsMetricbeatMonitoring(t *testing.T) {
	var o *Kibana

	// With default values
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Monitoring: KibanaMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring())

	// When disabled
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Monitoring: KibanaMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: false,
				},
			},
			Deployment: KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled but replica is set to 0
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KibanaSpec{
			Monitoring: KibanaMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: KibanaDeploymentSpec{
				Replicas: 0,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())
}
