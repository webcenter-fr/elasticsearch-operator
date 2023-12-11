package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetricbeatGetStatus(t *testing.T) {
	status := MetricbeatStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

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
				Persistence: &shared.DeploymentPersistenceSpec{},
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
				Persistence: &shared.DeploymentPersistenceSpec{
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
				Persistence: &shared.DeploymentPersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())

}

func TestMetricbeatIsPdb(t *testing.T) {
	var o Metricbeat

	// When default
	o = Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: MetricbeatSpec{},
	}
	assert.False(t, o.IsPdb())

	// When default with replica > 1
	o = Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: MetricbeatSpec{
			Deployment: MetricbeatDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 2,
				},
			},
		},
	}
	assert.True(t, o.IsPdb())

	// When PDB is set
	o = Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: MetricbeatSpec{
			Deployment: MetricbeatDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb())

}
