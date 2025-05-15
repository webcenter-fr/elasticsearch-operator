package logstash

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PodMonitorCondition shared.ConditionName = "PodMonitorReady"
	PodMonitorPhase     shared.PhaseName     = "PodMonitor"
)

type podMonitorReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *monitoringv1.PodMonitor]
}

func newPodMonitorReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *monitoringv1.PodMonitor]) {
	return &podMonitorReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *monitoringv1.PodMonitor](
			client,
			PodMonitorPhase,
			PodMonitorCondition,
			recorder,
		),
	}
}

// Read existing podMonitor
func (r *podMonitorReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*monitoringv1.PodMonitor], res reconcile.Result, err error) {
	pm := &monitoringv1.PodMonitor{}
	read = multiphase.NewMultiPhaseRead[*monitoringv1.PodMonitor]()

	// Read current podMonitor
	// We read it always, even if disabled to delete it
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetPodMonitorName(o)}, pm); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read podMonitor")
		}
		pm = nil
	}
	if pm != nil {
		read.AddCurrentObject(pm)
	}

	// Generate expected podMonitor
	expectedPm, err := buildPodMonitors(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate podMonitor")
	}
	read.SetExpectedObjects(expectedPm)

	return read, res, nil
}
