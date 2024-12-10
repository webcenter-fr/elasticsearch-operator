package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	appv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ExporterCondition shared.ConditionName = "ExporterReady"
	ExporterPhase     shared.PhaseName     = "Exporter"
)

type exporterReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newExporterReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &exporterReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			ExporterPhase,
			ExporterCondition,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
	}
}

// Read existing deployement
func (r *exporterReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	dpl := &appv1.Deployment{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current deployment
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetExporterDeployementName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read exporter deployment")
		}
		dpl = nil
	}
	if dpl != nil {
		read.SetCurrentObjects([]client.Object{dpl})
	}

	// Generate expected deployement
	expectedExporters, err := buildDeploymentExporters(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate exporter deployment")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedExporters))

	return read, res, nil
}
