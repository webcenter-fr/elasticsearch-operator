package elasticsearchapi

import (
	"context"
	"time"

	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type watchReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler]
	name string
}

func newWatchReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler] {
	return &watchReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *watchReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.Watch, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
	esClient, err := GetElasticsearchHandler(ctx, o, o.Spec.ElasticsearchRef, h.Client(), logger)
	if err != nil && o.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if o.DeletionTimestamp.IsZero() {
			return nil, reconcile.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newWatchApiClient(esClient)

	return handler, res, nil
}
