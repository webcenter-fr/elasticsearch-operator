package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildPodDisruptionBudget(t *testing.T) {

	var (
		err error
		pdb *policyv1.PodDisruptionBudget
		o   *kibanacrd.Kibana
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	pdb, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/pdb_default.yaml", pdb, test.CleanApi)

	// When Pdb is defined
	minUnavailable := intstr.FromInt(0)
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Deployment: kibanacrd.DeploymentSpec{
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
