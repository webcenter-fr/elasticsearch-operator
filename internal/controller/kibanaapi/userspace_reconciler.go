package kibanaapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type userSpaceReconciler struct {
	controller.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler]
	name string
}

func newUserSpaceReconciler(name string, client client.Client, recorder record.EventRecorder) controller.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler] {
	return &userSpaceReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *userSpaceReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject, logger *logrus.Entry) (handler controller.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler], res ctrl.Result, err error) {
	space := o.(*kibanaapicrd.UserSpace)
	kbClient, err := GetKibanaHandler(ctx, space, space.Spec.KibanaRef, h.Client(), logger)
	if err != nil && space.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Kibana not ready
	if kbClient == nil {
		if space.DeletionTimestamp.IsZero() {
			return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newUserSpaceApiClient(kbClient)

	return handler, res, nil
}

func (h *userSpaceReconciler) Create(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler], object *kbapi.KibanaSpace, logger *logrus.Entry) (res ctrl.Result, err error) {
	res, err = h.RemoteReconcilerAction.Create(ctx, o, data, handler, object, logger)
	if err != nil {
		return res, err
	}

	// Copy object that not enforce reconcile
	space := o.(*kibanaapicrd.UserSpace)

	for _, copySpec := range space.Spec.KibanaUserSpaceCopies {
		if !copySpec.IsForceUpdate() {
			cs := &kbapi.KibanaSpaceCopySavedObjectParameter{
				Spaces:            []string{space.GetExternalName()},
				IncludeReferences: copySpec.IsIncludeReference(),
				Overwrite:         copySpec.IsOverwrite(),
				CreateNewCopies:   copySpec.IsCreateNewCopy(),
				Objects:           make([]kbapi.KibanaSpaceObjectParameter, 0, len(copySpec.KibanaObjects)),
			}

			for _, kibanaObject := range copySpec.KibanaObjects {
				cs.Objects = append(cs.Objects, kbapi.KibanaSpaceObjectParameter{
					Type: kibanaObject.Type,
					ID:   kibanaObject.ID,
				})
			}

			if err = handler.Client().UserSpaceCopyObject(copySpec.OriginUserSpace, cs); err != nil {
				return res, errors.Wrap(err, "Error when copy objects on new Kibana user space")
			}
		}
	}

	return res, nil
}
