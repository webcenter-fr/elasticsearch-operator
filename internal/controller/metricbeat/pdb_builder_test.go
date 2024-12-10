package metricbeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildPodDisruptionBudget(t *testing.T) {
	var (
		err  error
		pdbs []policyv1.PodDisruptionBudget
		o    *beatcrd.Metricbeat
	)

	// With default values
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Empty(t, pdbs)

	// With default values with replica > 0
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{
			Deployment: beatcrd.MetricbeatDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 2,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*policyv1.PodDisruptionBudget](t, "testdata/pdb_default.yaml", &pdbs[0], scheme.Scheme)

	// When Pdb is defined
	minUnavailable := intstr.FromInt(0)
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{
			Deployment: beatcrd.MetricbeatDeploymentSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
					MinAvailable:   &minUnavailable,
					MaxUnavailable: nil,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*policyv1.PodDisruptionBudget](t, "testdata/pdb_with_global_spec.yaml", &pdbs[0], scheme.Scheme)
}
