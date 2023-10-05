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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type snapshotLifecyclePolicyReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec]
	controller.BaseReconciler
}

func newSnapshotLifecyclePolicyReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec] {
	return &snapshotLifecyclePolicyReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec](
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

func (h *snapshotLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec], res ctrl.Result, err error) {
	slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)
	esClient, err := GetElasticsearchHandler(ctx, slm, slm.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && slm.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newSnapshotLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}

func (h *snapshotLifecyclePolicyReconciler) Create(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec], object *eshandler.SnapshotLifecyclePolicySpec) (res ctrl.Result, err error) {

	// Check if snapshot repository exist befire create SLM
	if err = h.RemoteReconcilerAction.Custom(ctx, o, data, handler, object, func(handler any) error {
		client := handler.(eshandler.ElasticsearchHandler)
		slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

		repo, err := client.SnapshotRepositoryGet(slm.Spec.Repository)
		if err != nil {
			return errors.Wrap(err, "Error when get snapshot repository to check if exist before create SLM policy")
		}
		if repo == nil {
			return errors.Errorf("Snapshot repository %s not yet exist, skip it", slm.Spec.Repository)
		}

		return nil

	}); err != nil {
		h.Log.Warn(err.Error())
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return h.RemoteReconcilerAction.Create(ctx, o, data, handler, object)
}
