package metricbeat

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	o := resource.(*beatcrd.Metricbeat)
	pdb := &policyv1.PodDisruptionBudget{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current pdb
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetPDBName(o)}, pdb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read PDB")
		}
		pdb = nil
	}
	if pdb != nil {
		read.SetCurrentObjects([]client.Object{pdb})
	}

	// Generate expected pdb
	expectedPdbs, err := buildPodDisruptionBudgets(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate pdb")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedPdbs))

	return read, res, nil
}

func (r *pdbReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnorePDBSelector(),
	}
}
