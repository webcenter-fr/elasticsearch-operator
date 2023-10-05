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

type indexLifecyclePolicyReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse]
	controller.BaseReconciler
}

func newIndexLifecyclePolicyReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse] {
	return &indexLifecyclePolicyReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse](
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

func (h *indexLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse], res ctrl.Result, err error) {
	ilm := o.(*elasticsearchapicrd.IndexLifecyclePolicy)
	esClient, err := GetElasticsearchHandler(ctx, ilm, ilm.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && ilm.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newIndexLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}
