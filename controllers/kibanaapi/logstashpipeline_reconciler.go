package kibanaapi

import (
	"context"
	"time"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type logstashPipelineReconciler struct {
	controller.RemoteReconcilerAction[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler]
	controller.BaseReconciler
}

func newLogstashPipelineReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler] {
	return &logstashPipelineReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler](
			client,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Log:      logger,
			Recorder: recorder,
		},
	}
}

func (h *logstashPipelineReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler], res ctrl.Result, err error) {
	pipeline := o.(*kibanaapicrd.LogstashPipeline)
	kbClient, err := GetKibanaHandler(ctx, pipeline, pipeline.Spec.KibanaRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && pipeline.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Kibana not ready
	if kbClient == nil {
		if pipeline.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newLogstashPipelineApiClient(kbClient)

	return handler, res, nil
}
