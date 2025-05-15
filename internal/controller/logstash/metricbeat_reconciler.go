package logstash

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	MetricbeatCondition shared.ConditionName = "MetricbeatReady"
	MetricbeatPhase     shared.PhaseName     = "Metricbeat"
)

type metricbeatReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *beatcrd.Metricbeat]
}

func newMetricbeatReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *beatcrd.Metricbeat]) {
	return &metricbeatReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *beatcrd.Metricbeat](
			client,
			MetricbeatPhase,
			MetricbeatCondition,
			recorder,
		),
	}
}

// Read existing Metricbeat
func (r *metricbeatReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*beatcrd.Metricbeat], res reconcile.Result, err error) {
	metricbeat := &beatcrd.Metricbeat{}
	read = multiphase.NewMultiPhaseRead[*beatcrd.Metricbeat]()

	// Read current metricbeat
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetMetricbeatName(o)}, metricbeat); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read metricbeat")
		}
		metricbeat = nil
	}
	if metricbeat != nil {
		read.AddCurrentObject(metricbeat)
	}

	// Generate expected metricbeat
	expectedMetricbeats, err := buildMetricbeats(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate metricbeat")
	}
	read.SetExpectedObjects(expectedMetricbeats)

	return read, res, nil
}
