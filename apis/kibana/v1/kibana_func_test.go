package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetStatus(t *testing.T) {
	status := KibanaStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestIsPdb(t *testing.T) {
	var o Kibana

	// When default
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{},
	}
	assert.False(t, o.IsPdb())

	// When default with replica > 1
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			Deployment: KibanaDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 2,
				},
			},
		},
	}
	assert.True(t, o.IsPdb())

	// When PDB is set
	o = Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			Deployment: KibanaDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb())

}
