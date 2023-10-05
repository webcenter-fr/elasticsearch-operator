package elasticsearchapi

import (
	"context"
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type roleMappingReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping]
	controller.BaseReconciler
}

func newRoleMappingReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping] {
	return &roleMappingReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping](
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

func (h *roleMappingReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping], res ctrl.Result, err error) {
	rm := o.(*elasticsearchapicrd.RoleMapping)
	esClient, err := GetElasticsearchHandler(ctx, rm, rm.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && rm.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newRoleMappingApiClient(esClient)

	return handler, res, nil
}
