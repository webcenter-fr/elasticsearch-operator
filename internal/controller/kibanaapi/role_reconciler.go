package kibanaapi

import (
	"context"
	"time"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type roleReconciler struct {
	remote.RemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler]
	name string
}

func newRoleReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler] {
	return &roleReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *roleReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *kibanaapicrd.Role, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler], res reconcile.Result, err error) {
	kbClient, err := GetKibanaHandler(ctx, o, o.Spec.KibanaRef, h.Client(), logger)
	if err != nil && o.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Kibana not ready
	if kbClient == nil {
		if o.DeletionTimestamp.IsZero() {
			return nil, reconcile.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newRoleApiClient(kbClient)

	return handler, res, nil
}
