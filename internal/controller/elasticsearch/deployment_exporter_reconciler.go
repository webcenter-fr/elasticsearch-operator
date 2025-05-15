package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	appv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ExporterCondition shared.ConditionName = "ExporterReady"
	ExporterPhase     shared.PhaseName     = "Exporter"
)

type exporterReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.Deployment]
}

func newExporterReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.Deployment]) {
	return &exporterReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.Deployment](
			client,
			ExporterPhase,
			ExporterCondition,
			recorder,
		),
	}
}

// Read existing deployement
func (r *exporterReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*appv1.Deployment], res reconcile.Result, err error) {
	dpl := &appv1.Deployment{}
	read = multiphase.NewMultiPhaseRead[*appv1.Deployment]()

	// Read current deployment
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetExporterDeployementName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read exporter deployment")
		}
		dpl = nil
	}
	if dpl != nil {
		read.AddCurrentObject(dpl)
	}

	// Generate expected deployement
	expectedExporters, err := buildDeploymentExporters(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate exporter deployment")
	}
	read.SetExpectedObjects(expectedExporters)

	return read, res, nil
}
