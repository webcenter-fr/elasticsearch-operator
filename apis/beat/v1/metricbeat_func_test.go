package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetricbeatIsPersistence(t *testing.T) {
	var o *Metricbeat

	// With default value
	o = &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: MetricbeatSpec{},
	}

	assert.False(t, o.IsPersistence())

	// When persistence is not enabled
	o = &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: MetricbeatSpec{
			Deployment: MetricbeatDeploymentSpec{
				Persistence: &MetricbeatPersistenceSpec{},
			},
		},
	}

	assert.False(t, o.IsPersistence())

	// When claim PVC is set
	o = &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: MetricbeatSpec{
			Deployment: MetricbeatDeploymentSpec{
				Persistence: &MetricbeatPersistenceSpec{
					VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

	// When volume is set
	o = &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: MetricbeatSpec{
			Deployment: MetricbeatDeploymentSpec{
				Persistence: &MetricbeatPersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}
