package elasticsearchapi

import (
	"context"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type snapshotRepositoryReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler]
	name string
}

func newSnapshotRepositoryReconciler(name string, client client.Client, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler] {
	return &snapshotRepositoryReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *snapshotRepositoryReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject, logger *logrus.Entry) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	sr := o.(*elasticsearchapicrd.SnapshotRepository)
	esClient, err := GetElasticsearchHandler(ctx, sr, sr.Spec.ElasticsearchRef, h.Client(), logger)
	if err != nil && sr.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if sr.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newSnapshotRepositoryApiClient(esClient)

	return handler, res, nil
}
