package elasticsearchapi

import (
	"context"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type watchReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler]
	name string
}

func newWatchReconciler(name string, client client.Client, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler] {
	return &watchReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *watchReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject, logger *logrus.Entry) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	watch := o.(*elasticsearchapicrd.Watch)
	esClient, err := GetElasticsearchHandler(ctx, watch, watch.Spec.ElasticsearchRef, h.Client(), logger)
	if err != nil && watch.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if watch.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newWatchApiClient(esClient)

	return handler, res, nil
}
