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

type roleMappingReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler]
	name string
}

func newRoleMappingReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler] {
	return &roleMappingReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *roleMappingReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.RoleMapping, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newRoleMappingApiClient(esClient)

	return handler, res, nil
}
