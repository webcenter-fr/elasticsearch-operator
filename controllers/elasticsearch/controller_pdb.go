package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PdbCondition common.ConditionName = "PodDisruptionBudgetReady"
	PdbPhase     common.PhaseName     = "PodDisruptionBudget"
)

type PdbReconciler struct {
	common.Reconciler
}

func NewPdbReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &PdbReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "pdb",
			}),
			Name:   "pdb",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *PdbReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, PdbCondition, PdbPhase)
}

// Read existing pdbs
func (r *PdbReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	pdbList := &policyv1.PodDisruptionBudgetList{}

	// Read current node group pdbs
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, pdbList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read PDB")
	}
	data["currentObjects"] = pdbList.Items

	// Generate expected node group pdbs
	expectedPdbs, err := BuildPodDisruptionBudget(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate pdbs")
	}
	data["expectedObjects"] = expectedPdbs

	return res, nil
}

// Diff permit to check if pdbs are up to date
func (r *PdbReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdListDiff(ctx, resource, data, patch.IgnorePDBSelector())
}

// OnError permit to set status condition on the right state and record error
func (r *PdbReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, PdbCondition, PdbPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *PdbReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, PdbCondition, PdbPhase)
}
