package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetStatus(t *testing.T) {
	status := LogstashStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
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
				Persistence: &shared.DeploymentPersistenceSpec{},
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
				Persistence: &shared.DeploymentPersistenceSpec{
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
				Persistence: &shared.DeploymentPersistenceSpec{
					Volume: &v1.VolumeSource{},
				},
			},
		},
	}

	assert.True(t, o.IsPersistence())
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
				Deployment: shared.Deployment{
					Replicas: 2,
				},
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

func TestLogstashPkiSpecIsEnabled(t *testing.T) {
	var o LogstashPkiSpec

	// With default value
	assert.True(t, o.IsEnabled())

	// When enabled
	o.Enabled = ptr.To[bool](true)
	assert.True(t, o.IsEnabled())

	// When disabled
	o.Enabled = ptr.To[bool](false)
	assert.False(t, o.IsEnabled())
}

func TestLogstashPkiSpecHasBeatCertificate(t *testing.T) {
	var o LogstashPkiSpec

	// With default value
	assert.False(t, o.HasBeatCertificate())

	// When have beat certicate
	o.Tls = map[string]LogstashTlsSpec{
		"filebeat.crt": {
			Consumer: "beat",
		},
	}
	assert.True(t, o.HasBeatCertificate())
}
