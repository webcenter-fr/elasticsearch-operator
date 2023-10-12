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

type snapshotRepositoryReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler]
	controller.BaseReconciler
}

func newSnapshotRepositoryReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler] {
	return &snapshotRepositoryReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](
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

func (h *snapshotRepositoryReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	sr := o.(*elasticsearchapicrd.SnapshotRepository)
	esClient, err := GetElasticsearchHandler(ctx, sr, sr.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && sr.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newSnapshotRepositoryApiClient(esClient)

	return handler, res, nil
}
