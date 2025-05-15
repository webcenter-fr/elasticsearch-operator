package kibanaapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type userSpaceReconciler struct {
	remote.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler]
	name string
}

func newUserSpaceReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler] {
	return &userSpaceReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *userSpaceReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *kibanaapicrd.UserSpace, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler], res reconcile.Result, err error) {
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

	handler = newUserSpaceApiClient(kbClient)

	return handler, res, nil
}

func (h *userSpaceReconciler) Create(ctx context.Context, o *kibanaapicrd.UserSpace, data map[string]any, handler remote.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler], object *kbapi.KibanaSpace, logger *logrus.Entry) (res reconcile.Result, err error) {
	res, err = h.RemoteReconcilerAction.Create(ctx, o, data, handler, object, logger)
	if err != nil {
		return res, err
	}
	// Copy object
	for _, copySpec := range o.Spec.KibanaUserSpaceCopies {
		if !copySpec.IsForceUpdate() {
			cs := &kbapi.KibanaSpaceCopySavedObjectParameter{
				Spaces:            []string{o.GetExternalName()},
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
