package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildPodDisruptionBudget(t *testing.T) {

	var (
		err  error
		pdbs []policyv1.PodDisruptionBudget
		o    *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Empty(t, pdbs)

	// When pdb spec not provided, default
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Empty(t, pdbs)

	// When pdb spec not provided, default and replica > 0
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 2,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_default.yaml", &pdbs[0], test.CleanApi)

	// When Pdb is defined on global
	minUnavailable := intstr.FromInt(0)
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
				},
			},
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
					MinAvailable:   &minUnavailable,
					MaxUnavailable: nil,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_with_global_spec.yaml", &pdbs[0], test.CleanApi)

	// When Pdb is defined on nodeGroup
	minUnavailable = intstr.FromInt(10)
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 1,
					PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
						MinAvailable:   &minUnavailable,
						MaxUnavailable: nil,
					},
				},
			},
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{
					MinAvailable:   &minUnavailable,
					MaxUnavailable: nil,
				},
			},
		},
	}

	pdbs, err = buildPodDisruptionBudgets(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pdbs))
	test.EqualFromYamlFile(t, "testdata/pdb_with_global_and_local_spec.yaml", &pdbs[0], test.CleanApi)

}
