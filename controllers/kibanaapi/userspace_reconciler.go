package kibanaapi

import (
	"context"

	"emperror.dev/errors"
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

type userSpaceReconciler struct {
	controller.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace]
	controller.BaseReconciler
}

func newUserSpaceReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace] {
	return &userSpaceReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace](
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

func (h *userSpaceReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace], res ctrl.Result, err error) {
	space := o.(*kibanaapicrd.UserSpace)
	kbClient, err := GetKibanaHandler(ctx, space, space.Spec.KibanaRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && space.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	handler = newUserSpaceApiClient(kbClient)

	return handler, res, nil
}

func (h *userSpaceReconciler) Create(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace], object *kbapi.KibanaSpace) (res ctrl.Result, err error) {
	res, err = h.RemoteReconcilerAction.Create(ctx, o, data, handler, object)
	if err != nil {
		return res, err
	}

	// Copy object that not enforce reconcile
	if err = h.RemoteReconcilerAction.Custom(ctx, o, data, handler, object, func(handler any) error {
		client := handler.(kbhandler.KibanaHandler)
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

				if err = client.UserSpaceCopyObject(copySpec.OriginUserSpace, cs); err != nil {
					return errors.Wrap(err, "Error when copy objects on new Kibana user space")
				}
			}
		}

		return nil

	}); err != nil {
		return res, err
	}

	return res, nil

}
