package elasticsearchapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type snapshotLifecyclePolicyReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler]
	controller.BaseReconciler
}

func newSnapshotLifecyclePolicyReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler] {
	return &snapshotLifecyclePolicyReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler](
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

func (h *snapshotLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)
	esClient, err := GetElasticsearchHandler(ctx, slm, slm.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && slm.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if slm.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newSnapshotLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}

func (h *snapshotLifecyclePolicyReconciler) Create(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], object *eshandler.SnapshotLifecyclePolicySpec) (res ctrl.Result, err error) {
	// Check if snapshot repository exist befire create SLM

	slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	repo, err := handler.Client().SnapshotRepositoryGet(slm.Spec.Repository)
	if err != nil {
		return res, errors.Wrap(err, "Error when get snapshot repository to check if exist before create SLM policy")
	}
	if repo == nil {
		return res, errors.Errorf("Snapshot repository %s not yet exist, skip it", slm.Spec.Repository)
	}

	return h.RemoteReconcilerAction.Create(ctx, o, data, handler, object)
}
