package kibanaapi

import (
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
)

type userSpaceApiClient struct {
	controller.BasicRemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace]
	client kbhandler.KibanaHandler
}

func newUserSpaceApiClient(client kbhandler.KibanaHandler) controller.RemoteExternalReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace] {
	return &userSpaceApiClient{
		client: client,
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

func (h *userSpaceApiClient) Get(name string) (object *kbapi.KibanaSpace, err error) {
	return h.client.UserSpaceGet(name)
}

func (h *userSpaceApiClient) Create(object *kbapi.KibanaSpace) (err error) {
	return h.client.UserSpaceCreate(object)
}

func (h *userSpaceApiClient) Update(object *kbapi.KibanaSpace) (err error) {
	return h.client.UserSpaceUpdate(object)

}

func (h *userSpaceApiClient) Delete(name string) (err error) {
	return h.client.UserSpaceDelete(name)

}

func (h *userSpaceApiClient) Diff(currentOject *kbapi.KibanaSpace, expectedObject *kbapi.KibanaSpace, originalObject *kbapi.KibanaSpace, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.client.UserSpaceDiff(currentOject, expectedObject, originalObject)
}

func (h *userSpaceApiClient) Custom(f func(handler any) error) (err error) {
	return f(h.client)
}
