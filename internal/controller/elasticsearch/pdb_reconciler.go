package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PdbCondition shared.ConditionName = "PodDisruptionBudgetReady"
	PdbPhase     shared.PhaseName     = "PodDisruptionBudget"
)

type pdbReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newPdbReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &pdbReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			PdbPhase,
			PdbCondition,
			recorder,
		),
	}
}

// Read existing pdbs
func (r *pdbReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	pdbList := &policyv1.PodDisruptionBudgetList{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current node group pdbs
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, pdbList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read PDB")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(pdbList.Items))

	// Generate expected node group pdbs
	expectedPdbs, err := buildPodDisruptionBudgets(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate pdbs")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedPdbs))

	return read, res, nil
}

func (r *pdbReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnorePDBSelector(),
	}
}
