package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsPrometheusMonitoring(t *testing.T) {
	var o *Logstash

	// With default values
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{},
	}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Monitoring: LogstashMonitoringSpec{
				Prometheus: &LogstashPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Monitoring: LogstashMonitoringSpec{
				Prometheus: &LogstashPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsPersistence(t *testing.T) {
	var o *Logstash

	// With default value
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{},
	}

	assert.False(t, o.IsPersistence())

	// When persistence is not enabled
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Deployment: LogstashDeploymentSpec{
				Persistence: &LogstashPersistenceSpec{},
			},
		},
	}

	assert.False(t, o.IsPersistence())

	// When claim PVC is set
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Deployment: LogstashDeploymentSpec{
				Persistence: &LogstashPersistenceSpec{
					VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

	// When volume is set
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Deployment: LogstashDeploymentSpec{
				Persistence: &LogstashPersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}

func TestIsMetricbeatMonitoring(t *testing.T) {
	var o *Logstash

	// With default values
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Monitoring: LogstashMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: LogstashDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring())

	// When disabled
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Monitoring: LogstashMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: false,
				},
			},
			Deployment: LogstashDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled but replicas is set to 0
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Monitoring: LogstashMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: LogstashDeploymentSpec{
				Replicas: 0,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())
}

func TestIsPdb(t *testing.T) {
	var o Logstash

	// When default
	o = Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: LogstashSpec{},
	}
	assert.False(t, o.IsPdb())

	// When default with replica > 1
	o = Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: LogstashSpec{
			Deployment: LogstashDeploymentSpec{
				Replicas: 2,
			},
		},
	}
	assert.True(t, o.IsPdb())

	// When PDB is set
	o = Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: LogstashSpec{
			Deployment: LogstashDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb())

}
