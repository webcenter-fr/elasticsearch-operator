package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
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
	ServiceCondition = "ServiceReady"
	ServicePhase     = "Service"
)

type ServiceReconciler struct {
	common.Reconciler
}

func NewServiceReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &ServiceReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "service",
			}),
			Name:   "service",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *ServiceReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ServiceCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ServiceCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = ServicePhase

	return res, nil
}

// Read existing services
func (r *ServiceReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	service := &corev1.Service{}

	// Read current service
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetServiceName(o)}, service); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read service")
		}
		service = nil
	}

	data["currentObject"] = service

	// Generate expected service
	expectedService, err := BuildService(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate service")
	}
	data["expectedObject"] = expectedService

	return res, nil
}

// Diff permit to check if services are up to date
func (r *ServiceReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *ServiceReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	return r.StdOnError(ctx, resource, data, currentErr, &o.Status.Conditions, ServiceCondition)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ServiceReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Service successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ServiceCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    ServiceCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
