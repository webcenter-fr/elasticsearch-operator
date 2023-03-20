package metricbeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildPodDisruptionBudget(t *testing.T) {

	var (
		err error
		pdb *policyv1.PodDisruptionBudget
		o   *beatcrd.Metricbeat
	)

	// With default values
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	pdb, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/pdb_default.yaml", pdb, test.CleanApi)

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

	pdb, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/pdb_with_global_spec.yaml", pdb, test.CleanApi)

}