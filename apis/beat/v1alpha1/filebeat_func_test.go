package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsPrometheusMonitoring(t *testing.T) {
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
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
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
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsPersistence(t *testing.T) {
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
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{},
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
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{
					VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}

	// When volume is set
	o = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: FilebeatSpec{
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}

func TestIsManaged(t *testing.T) {
	var o LogstashRef

	// When managed
	o = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{
			Name:          "test",
			TargetService: "beat",
		},
	}
	assert.True(t, o.IsManaged())

	// When not managed
	o = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{},
	}
	assert.False(t, o.IsManaged())

	o = LogstashRef{}
	assert.False(t, o.IsManaged())

}

func TestIsExternal(t *testing.T) {
	var o LogstashRef

	// When external
	o = LogstashRef{
		ExternalLogstashRef: &LogstashExternalRef{
			Addresses: []string{
				"test",
			},
		},
	}
	assert.True(t, o.IsExternal())

	// When not managed
	o = LogstashRef{
		ExternalLogstashRef: &LogstashExternalRef{},
	}
	assert.False(t, o.IsExternal())

	o = LogstashRef{}
	assert.False(t, o.IsExternal())

}
