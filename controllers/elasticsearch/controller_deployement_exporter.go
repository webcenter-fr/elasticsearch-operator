package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ExporterCondition = "ExporterReady"
	ExporterPhase     = "Exporter"
)

type ExporterReconciler struct {
	common.Reconciler
}

func NewExporterReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &ExporterReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "exporter",
			}),
			Name:   "exporter",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *ExporterReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ExporterCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ExporterCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = ExporterPhase

	return res, nil
}

// Read existing deployement
func (r *ExporterReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	dpl := &appv1.Deployment{}

	// Read current deployment
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetExporterDeployementName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read exporter deployment")
		}
		dpl = nil
	}
	data["currentObject"] = dpl

	// Generate expected deployement
	expectedExporter, err := BuildDeploymentExporter(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate exporter deployment")
	}
	data["expectedObject"] = expectedExporter

	return res, nil
}

// Diff permit to check if deployment is up to date
func (r *ExporterReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *ExporterReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    ExporterCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ExporterReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Exporter deployment successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ExporterCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    ExporterCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
