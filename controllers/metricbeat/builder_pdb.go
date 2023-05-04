package metricbeat

import (
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GeneratePodDisruptionBudget permit to generate pod disruption budgets
func BuildPodDisruptionBudget(mb *beatcrd.Metricbeat) (pdb *policyv1.PodDisruptionBudget, err error) {

	maxUnavailable := intstr.FromInt(1)
	pdb = &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mb.Namespace,
			Name:        GetPDBName(mb),
			Labels:      getLabels(mb),
			Annotations: getAnnotations(mb),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                       mb.Name,
					beatcrd.MetricbeatAnnotationKey: "true",
				},
			},
		},
	}

	// Merge with specified
	if err = k8sbuilder.MergeK8s(&pdb.Spec, pdb.Spec, mb.Spec.Deployment.PodDisruptionBudgetSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge expected PDB with global PDB")
	}
	if pdb.Spec.MinAvailable == nil && pdb.Spec.MaxUnavailable == nil {
		pdb.Spec.MaxUnavailable = &maxUnavailable
	}

	return pdb, nil
}
