package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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
