package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestFilebeatGetStatus(t *testing.T) {
	status := FilebeatStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
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
				Persistence: &shared.DeploymentPersistenceSpec{},
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
				Persistence: &shared.DeploymentPersistenceSpec{
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
				Persistence: &shared.DeploymentPersistenceSpec{
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
				Deployment: shared.Deployment{
					Replicas: 2,
				},
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

func TestFilebeatPkiSpecIsEnabled(t *testing.T) {
	var o FilebeatPkiSpec

	// With default value
	assert.True(t, o.IsEnabled())

	// When enabled
	o.Enabled = ptr.To[bool](true)
	assert.True(t, o.IsEnabled())

	// When disabled
	o.Enabled = ptr.To[bool](false)
	assert.False(t, o.IsEnabled())
}
