package kibana

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GeneratePodDisruptionBudget permit to generate pod disruption budgets for each node group
func buildPodDisruptionBudgets(kb *kibanacrd.Kibana) (pdbs []policyv1.PodDisruptionBudget, err error) {
	if !kb.IsPdb() {
		return nil, nil
	}

	pdbs = make([]policyv1.PodDisruptionBudget, 0, 1)

	maxUnavailable := intstr.FromInt(1)
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetPDBName(kb),
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                     kb.Name,
					kibanacrd.KibanaAnnotationKey: "true",
				},
			},
		},
	}

	// Merge with specified
	if err = k8sbuilder.MergeK8s(&pdb.Spec, pdb.Spec, kb.Spec.Deployment.PodDisruptionBudgetSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge expected PDB with global PDB")
	}
	if pdb.Spec.MinAvailable == nil && pdb.Spec.MaxUnavailable == nil {
		pdb.Spec.MaxUnavailable = &maxUnavailable
	}

	pdbs = append(pdbs, *pdb)

	return pdbs, nil
}
