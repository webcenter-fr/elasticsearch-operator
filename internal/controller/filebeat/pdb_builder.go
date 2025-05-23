package filebeat

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GeneratePodDisruptionBudget permit to generate pod disruption budgets
func buildPodDisruptionBudgets(fb *beatcrd.Filebeat) (pdbs []*policyv1.PodDisruptionBudget, err error) {
	if !fb.IsPdb() {
		return nil, nil
	}

	pdbs = make([]*policyv1.PodDisruptionBudget, 0, 1)

	maxUnavailable := intstr.FromInt(1)
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   fb.Namespace,
			Name:        GetPDBName(fb),
			Labels:      getLabels(fb),
			Annotations: getAnnotations(fb),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                     fb.Name,
					beatcrd.FilebeatAnnotationKey: "true",
				},
			},
		},
	}

	// Merge with specified
	if err = k8sbuilder.MergeK8s(&pdb.Spec, pdb.Spec, fb.Spec.Deployment.PodDisruptionBudgetSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge expected PDB with global PDB")
	}
	if pdb.Spec.MinAvailable == nil && pdb.Spec.MaxUnavailable == nil {
		pdb.Spec.MaxUnavailable = &maxUnavailable
	}

	pdbs = append(pdbs, pdb)

	return pdbs, nil
}
