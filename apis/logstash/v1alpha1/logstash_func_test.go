package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
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
			Monitoring: MonitoringSpec{
				Prometheus: &PrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsOneForEachLogstashInstance(t *testing.T) {
	var o *Logstash

	// When not set
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Ingresses: []Ingress{
				{
					Name: "test",
				},
			},
		},
	}
	assert.False(t, o.Spec.Ingresses[0].IsOneForEachLogstashInstance())

	// When set to false
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Ingresses: []Ingress{
				{
					Name:                       "test",
					OneForEachLogstashInstance: pointer.Bool(false),
				},
			},
		},
	}
	assert.False(t, o.Spec.Ingresses[0].IsOneForEachLogstashInstance())

	// When set to true
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Ingresses: []Ingress{
				{
					Name:                       "test",
					OneForEachLogstashInstance: pointer.Bool(true),
				},
			},
		},
	}
	assert.True(t, o.Spec.Ingresses[0].IsOneForEachLogstashInstance())

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
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{},
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
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{
					VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}

	// When volume is set
	o = &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
			Deployment: DeploymentSpec{
				Persistence: &PersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}
