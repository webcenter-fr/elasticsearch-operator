package elasticsearch

import (
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GeneratePodDisruptionBudget permit to generate pod disruption budgets for each node group
func BuildPodDisruptionBudget(es *elasticsearchapi.Elasticsearch) (podDisruptionBudgets []policyv1.PodDisruptionBudget, err error) {
	podDisruptionBudgets = make([]policyv1.PodDisruptionBudget, 0, len(es.Spec.NodeGroups))
	var (
		pdb *policyv1.PodDisruptionBudget
	)

	maxUnavailable := intstr.FromInt(1)
	for _, nodeGroup := range es.Spec.NodeGroups {
		pdb = &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetNodeGroupPDBName(es, nodeGroup.Name),
				Labels:      getLabels(es),
				Annotations: getAnnotations(es),
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":   es.Name,
						"nodeGroup": nodeGroup.Name,
					},
				},
			},
		}

		// Merge with specified
		if err = k8sbuilder.MergeK8s(&pdb.Spec, pdb.Spec, es.Spec.GlobalNodeGroup.PodDisruptionBudgetSpec); err != nil {
			return nil, errors.Wrap(err, "Error when merge expected PDB with global PDB")
		}
		if err = k8sbuilder.MergeK8s(&pdb.Spec, pdb.Spec, nodeGroup.PodDisruptionBudgetSpec); err != nil {
			return nil, errors.Wrap(err, "Error when merge expected PDB with node group PDB")
		}
		if pdb.Spec.MinAvailable == nil && pdb.Spec.MaxUnavailable == nil {
			pdb.Spec.MaxUnavailable = &maxUnavailable
		}

		podDisruptionBudgets = append(podDisruptionBudgets, *pdb)
	}

	return podDisruptionBudgets, nil
}
