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

type indexLifecyclePolicyReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler]
	name string
}

func newIndexLifecyclePolicyReconciler(name string, client client.Client, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler] {
	return &indexLifecyclePolicyReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *indexLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject, logger *logrus.Entry) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	ilm := o.(*elasticsearchapicrd.IndexLifecyclePolicy)
	esClient, err := GetElasticsearchHandler(ctx, ilm, ilm.Spec.ElasticsearchRef, h.Client(), logger)
	if err != nil && ilm.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if ilm.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newIndexLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}
