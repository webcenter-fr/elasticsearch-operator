package elasticsearchapi

import (
	"context"
	"time"

	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type indexLifecyclePolicyReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler]
	name string
}

func newIndexLifecyclePolicyReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler] {
	return &indexLifecyclePolicyReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *indexLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.IndexLifecyclePolicy, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newIndexLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}
