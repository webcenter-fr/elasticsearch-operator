package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PdbCondition shared.ConditionName = "PodDisruptionBudgetReady"
	PdbPhase     shared.PhaseName     = "PodDisruptionBudget"
)

type pdbReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *policyv1.PodDisruptionBudget]
}

func newPdbReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *policyv1.PodDisruptionBudget]) {
	return &pdbReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *policyv1.PodDisruptionBudget](
			client,
			PdbPhase,
			PdbCondition,
			recorder,
		),
	}
}

// Read existing pdbs
func (r *pdbReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*policyv1.PodDisruptionBudget], res reconcile.Result, err error) {
	pdbList := &policyv1.PodDisruptionBudgetList{}
	read = multiphase.NewMultiPhaseRead[*policyv1.PodDisruptionBudget]()

	// Read current node group pdbs
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, pdbList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read PDB")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(pdbList.Items))

	// Generate expected node group pdbs
	expectedPdbs, err := buildPodDisruptionBudgets(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate pdbs")
	}
	read.SetExpectedObjects(expectedPdbs)

	return read, res, nil
}

func (r *pdbReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnorePDBSelector(),
	}
}
