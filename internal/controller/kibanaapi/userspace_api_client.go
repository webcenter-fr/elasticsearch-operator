package kibanaapi

import (
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
)

type userSpaceApiClient struct {
	remote.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler]
}

func newUserSpaceApiClient(client kbhandler.KibanaHandler) remote.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler] {
	return &userSpaceApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler](client),
	}
}

func (h *userSpaceApiClient) Build(o *kibanaapicrd.UserSpace) (space *kbapi.KibanaSpace, err error) {
	space = &kbapi.KibanaSpace{
		ID:               o.GetExternalName(),
		Name:             o.Spec.Name,
		Description:      o.Spec.Description,
		DisabledFeatures: o.Spec.DisabledFeatures,
		Initials:         o.Spec.Initials,
		Color:            o.Spec.Color,
	}

	return space, nil
}

func (h *userSpaceApiClient) Get(o *kibanaapicrd.UserSpace) (object *kbapi.KibanaSpace, err error) {
	return h.Client().UserSpaceGet(o.GetExternalName())
}

func (h *userSpaceApiClient) Create(object *kbapi.KibanaSpace, o *kibanaapicrd.UserSpace) (err error) {
	return h.Client().UserSpaceCreate(object)
}

func (h *userSpaceApiClient) Update(object *kbapi.KibanaSpace, o *kibanaapicrd.UserSpace) (err error) {
	return h.Client().UserSpaceUpdate(object)
}

func (h *userSpaceApiClient) Delete(o *kibanaapicrd.UserSpace) (err error) {
	return h.Client().UserSpaceDelete(o.GetExternalName())
}

func (h *userSpaceApiClient) Diff(currentOject *kbapi.KibanaSpace, expectedObject *kbapi.KibanaSpace, originalObject *kbapi.KibanaSpace, o *kibanaapicrd.UserSpace, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().UserSpaceDiff(currentOject, expectedObject, originalObject)
}
