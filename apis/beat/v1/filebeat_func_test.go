package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilebeatIsPrometheusMonitoring(t *testing.T) {
	var o *Filebeat

	// With default values
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{},
	}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Monitoring: FilebeatMonitoringSpec{
				Prometheus: &FilebeatPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Monitoring: FilebeatMonitoringSpec{
				Prometheus: &FilebeatPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestFilebeatIsPersistence(t *testing.T) {
	var o *Filebeat

	// With default value
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{},
	}

	assert.False(t, o.IsPersistence())

	// When persistence is not enabled
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Deployment: FilebeatDeploymentSpec{
				Persistence: &FilebeatPersistenceSpec{},
			},
		},
	}

	assert.False(t, o.IsPersistence())

	// When claim PVC is set
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Deployment: FilebeatDeploymentSpec{
				Persistence: &FilebeatPersistenceSpec{
					VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

	// When volume is set
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Deployment: FilebeatDeploymentSpec{
				Persistence: &FilebeatPersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}

func TestFilebeatIsManaged(t *testing.T) {
	var o FilebeatLogstashRef

	// When managed
	o = FilebeatLogstashRef{
		ManagedLogstashRef: &FilebeatLogstashManagedRef{
			Name:          "test",
			TargetService: "beat",
		},
	}
	assert.True(t, o.IsManaged())

	// When not managed
	o = FilebeatLogstashRef{
		ManagedLogstashRef: &FilebeatLogstashManagedRef{},
	}
	assert.False(t, o.IsManaged())

	o = FilebeatLogstashRef{}
	assert.False(t, o.IsManaged())

}

func TestFilebeatIsExternal(t *testing.T) {
	var o FilebeatLogstashRef

	// When external
	o = FilebeatLogstashRef{
		ExternalLogstashRef: &FilebeatLogstashExternalRef{
			Addresses: []string{
				"test",
			},
		},
	}
	assert.True(t, o.IsExternal())

	// When not managed
	o = FilebeatLogstashRef{
		ExternalLogstashRef: &FilebeatLogstashExternalRef{},
	}
	assert.False(t, o.IsExternal())

	o = FilebeatLogstashRef{}
	assert.False(t, o.IsExternal())

}

func TestFilebeatIsMetricbeatMonitoring(t *testing.T) {
	var o *Filebeat

	// With default values
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Monitoring: FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: FilebeatDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring())

	// When disabled
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Monitoring: FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: false,
				},
			},
			Deployment: FilebeatDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())

	// When enabled but replica is set to 0
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Monitoring: FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
				},
			},
			Deployment: FilebeatDeploymentSpec{
				Replicas: 0,
			},
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring())
}

func TestFilebeatIsPdb(t *testing.T) {
	var o Filebeat

	// When default
	o = Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: FilebeatSpec{},
	}
	assert.False(t, o.IsPdb())

	// When default with replica > 1
	o = Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: FilebeatSpec{
			Deployment: FilebeatDeploymentSpec{
				Replicas: 2,
			},
		},
	}
	assert.True(t, o.IsPdb())

	// When PDB is set
	o = Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: FilebeatSpec{
			Deployment: FilebeatDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb())

}
