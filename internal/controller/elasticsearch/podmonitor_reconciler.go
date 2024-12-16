package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodMonitorCondition shared.ConditionName = "PodMonitorReady"
	PodMonitorPhase     shared.PhaseName     = "PodMonitor"
)

type podMonitorReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newPodMonitorReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &podMonitorReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			PodMonitorPhase,
			PodMonitorCondition,
			recorder,
		),
	}
}

// Read existing podMonitor
func (r *podMonitorReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	pm := &monitoringv1.PodMonitor{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current podMonitor if enabled
	if o.Spec.Monitoring.IsPrometheusMonitoring() {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetPodMonitorName(o)}, pm); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read podMonitor")
			}
			pm = nil
		}
	} else {
		pm = nil
	}
	if pm != nil {
		read.SetCurrentObjects([]client.Object{pm})
	}

	// Generate expected podMonitor
	expectedPm, err := buildPodMonitors(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate podMonitor")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedPm))

	return read, res, nil
}
