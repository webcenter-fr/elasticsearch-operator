package logstash

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PdbCondition shared.ConditionName = "PodDisruptionBudgetReady"
	PdbPhase     shared.PhaseName     = "PodDisruptionBudget"
)

type pdbReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *policyv1.PodDisruptionBudget]
}

func newPdbReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *policyv1.PodDisruptionBudget]) {
	return &pdbReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *policyv1.PodDisruptionBudget](
			client,
			PdbPhase,
			PdbCondition,
			recorder,
		),
	}
}

// Read existing pdbs
func (r *pdbReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*policyv1.PodDisruptionBudget], res reconcile.Result, err error) {
	pdb := &policyv1.PodDisruptionBudget{}
	read = multiphase.NewMultiPhaseRead[*policyv1.PodDisruptionBudget]()

	// Read current pdb
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetPDBName(o)}, pdb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read PDB")
		}
		pdb = nil
	}
	if pdb != nil {
		read.AddCurrentObject(pdb)
	}

	// Generate expected pdb
	expectedPdbs, err := buildPodDisruptionBudgets(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate pdb")
	}
	read.SetExpectedObjects(expectedPdbs)

	return read, res, nil
}

func (r *pdbReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnorePDBSelector(),
	}
}
