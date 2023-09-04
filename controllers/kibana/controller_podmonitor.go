package kibana

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodMonitorCondition common.ConditionName = "PodMonitorReady"
	PodMonitorPhase     common.PhaseName     = "PodMonitor"
)

type PodMonitorReconciler struct {
	common.Reconciler
}

func NewPodMonitorReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &PodMonitorReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "podMonitor",
			}),
			Name:   "podMonitor",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *PodMonitorReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, PodMonitorCondition, PodMonitorPhase)
}

// Read existing podMonitor
func (r *PodMonitorReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	pm := &monitoringv1.PodMonitor{}

	// Read current podMonitor if enabled
	if o.IsPrometheusMonitoring() {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetPodMonitorName(o)}, pm); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read podMonitor")
			}
			pm = nil
		}
	} else {
		pm = nil
	}

	data["currentObject"] = pm

	// Generate expected podMonitor
	expectedPm, err := BuildPodMonitor(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate podMonitor")
	}
	data["expectedObject"] = expectedPm

	return res, nil
}

// Diff permit to check if podMonitor is up to date
func (r *PodMonitorReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *PodMonitorReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, PodMonitorCondition, PodMonitorPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *PodMonitorReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, PodMonitorCondition, PodMonitorPhase)
}
