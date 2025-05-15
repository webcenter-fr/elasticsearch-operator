package elasticsearchapi

import (
	"context"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type roleReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler]
	name string
}

func newRoleReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler] {
	return &roleReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *roleReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.Role, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newRoleApiClient(esClient)

	return handler, res, nil
}
