package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildPodDisruptionBudget(t *testing.T) {

	var (
		err  error
		pdbs []policyv1.PodDisruptionBudget
		o    *kibanacrd.Kibana
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Empty(t, pdbs)

	// With default values with replica > 0
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 2,
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*policyv1.PodDisruptionBudget](t, "testdata/pdb_default.yaml", &pdbs[0], scheme.Scheme)

	// When Pdb is defined
	minUnavailable := intstr.FromInt(0)
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Deployment: kibanacrd.KibanaDeploymentSpec{
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