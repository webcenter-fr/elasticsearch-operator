package elasticsearchapi

import (
	"context"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type componentTemplateReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler]
	name string
}

func newComponentTemplateReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler] {
	return &componentTemplateReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *componentTemplateReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.ComponentTemplate, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newComponentTemplateApiClient(esClient)

	return handler, res, nil
}
