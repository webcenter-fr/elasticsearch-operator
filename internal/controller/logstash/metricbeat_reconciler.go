package logstash

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricbeatCondition shared.ConditionName = "MetricbeatReady"
	MetricbeatPhase     shared.PhaseName     = "Metricbeat"
)

type metricbeatReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newMetricbeatReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &metricbeatReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			MetricbeatPhase,
			MetricbeatCondition,
			recorder,
		),
	}
}

// Read existing Metricbeat
func (r *metricbeatReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)
	metricbeat := &beatcrd.Metricbeat{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current metricbeat
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetMetricbeatName(o)}, metricbeat); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read metricbeat")
		}
		metricbeat = nil
	}
	if metricbeat != nil {
		read.SetCurrentObjects([]client.Object{metricbeat})
	}

	// Generate expected metricbeat
	expectedMetricbeats, err := buildMetricbeats(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate metricbeat")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedMetricbeats))

	return read, res, nil
}
