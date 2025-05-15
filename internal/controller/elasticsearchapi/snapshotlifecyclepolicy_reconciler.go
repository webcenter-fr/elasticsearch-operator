package elasticsearchapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type snapshotLifecyclePolicyReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler]
	name string
}

func newSnapshotLifecyclePolicyReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler] {
	return &snapshotLifecyclePolicyReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *snapshotLifecyclePolicyReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.SnapshotLifecyclePolicy, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newSnapshotLifecyclePolicyApiClient(esClient)

	return handler, res, nil
}

func (h *snapshotLifecyclePolicyReconciler) Create(ctx context.Context, o *elasticsearchapicrd.SnapshotLifecyclePolicy, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], object *eshandler.SnapshotLifecyclePolicySpec, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Check if snapshot repository exist befire create SLM

	repo, err := handler.Client().SnapshotRepositoryGet(o.Spec.Repository)
	if err != nil {
		return res, errors.Wrap(err, "Error when get snapshot repository to check if exist before create SLM policy")
	}
	if repo == nil {
		return res, errors.Errorf("Snapshot repository %s not yet exist, skip it", o.Spec.Repository)
	}

	return h.RemoteReconcilerAction.Create(ctx, o, data, handler, object, logger)
}
