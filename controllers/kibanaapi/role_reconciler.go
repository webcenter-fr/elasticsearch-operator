package kibanaapi

import (
	"context"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type roleReconciler struct {
	controller.RemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler]
	controller.BaseReconciler
}

func newRoleReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler] {
	return &roleReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler](
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

func (h *roleReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler], res ctrl.Result, err error) {
	role := o.(*kibanaapicrd.Role)
	kbClient, err := GetKibanaHandler(ctx, role, role.Spec.KibanaRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && role.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	handler = newRoleApiClient(kbClient)

	return handler, res, nil
}
