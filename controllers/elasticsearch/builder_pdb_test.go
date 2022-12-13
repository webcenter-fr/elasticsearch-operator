package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildPodDisruptionBudget(t *testing.T) {

	var (
		err  error
		pdbs []policyv1.PodDisruptionBudget
		o    *elasticsearchapi.Elasticsearch
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{},
	}

	pdbs, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	assert.Empty(t, pdbs)

	// When pdb spec not provided, default
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
				},
			},
		},
	}

	pdbs, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_default.yaml", &pdbs[0], test.CleanApi)

	// When Pdb is defined on global
	minUnavailable := intstr.FromInt(0)
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
				},
			},
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
					MinAvailable:   &minUnavailable,
					MaxUnavailable: nil,
				},
			},
		},
	}

	pdbs, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_with_global_spec.yaml", &pdbs[0], test.CleanApi)

	// When Pdb is defined on nodeGroup
	minUnavailable = intstr.FromInt(10)
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
					PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
						MinAvailable:   &minUnavailable,
						MaxUnavailable: nil,
					},
				},
			},
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
					MinAvailable:   &minUnavailable,
					MaxUnavailable: nil,
				},
			},
		},
	}

	pdbs, err = BuildPodDisruptionBudget(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_with_global_and_local_spec.yaml", &pdbs[0], test.CleanApi)

}
