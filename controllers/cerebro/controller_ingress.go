package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IngressCondition common.ConditionName = "IngressReady"
	IngressPhase     common.PhaseName     = "Ingress"
)

type IngressReconciler struct {
	common.Reconciler
}

func NewIngressReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &IngressReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "ingress",
			}),
			Name:   "ingress",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *IngressReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, IngressCondition, IngressPhase)
}

// Read existing ingress
func (r *IngressReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	ingress := &networkingv1.Ingress{}

	// Read current ingress
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, ingress); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read ingress")
		}
		ingress = nil
	}
	data["currentObject"] = ingress

	// Generate expected ingress
	expectedIngress, err := BuildIngress(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate ingress")
	}
	data["expectedObject"] = expectedIngress

	return res, nil
}

// Diff permit to check if ingress is up to date
func (r *IngressReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *IngressReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, IngressCondition, IngressPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *IngressReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, IngressCondition, IngressPhase)
}
