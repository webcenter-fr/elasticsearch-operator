package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricbeatCondition common.ConditionName = "MetricbeatReady"
	MetricbeatPhase     common.PhaseName     = "Metricbeat"
)

type MetricbeatReconciler struct {
	common.Reconciler
}

func NewMetricbeatReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &MetricbeatReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "metricbeat",
			}),
			Name:   "metricbeat",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *MetricbeatReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, MetricbeatCondition, MetricbeatPhase)
}

// Read existing Metricbeat
func (r *MetricbeatReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	metricbeat := &beatcrd.Metricbeat{}

	// Read current metricbeat
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetMetricbeatName(o)}, metricbeat); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read metricbeat")
		}
		metricbeat = nil
	}
	data["currentObject"] = metricbeat

	// Generate expected metricbeat
	expectedMetricbeat, err := BuildMetricbeat(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate metricbeat")
	}
	data["expectedObject"] = expectedMetricbeat

	return res, nil
}

// Diff permit to check if metricbeat is up to date
func (r *MetricbeatReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *MetricbeatReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, MetricbeatCondition, MetricbeatPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *MetricbeatReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, MetricbeatCondition, MetricbeatPhase)
}
