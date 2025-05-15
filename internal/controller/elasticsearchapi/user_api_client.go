package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type userApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]
}

func newUserApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler] {
	return &userApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler](client),
	}
}

func (h *userApiClient) Build(o *elasticsearchapicrd.User) (user *olivere.XPackSecurityPutUserRequest, err error) {
	user = &olivere.XPackSecurityPutUserRequest{
		Enabled:      o.Spec.Enabled,
		Email:        o.Spec.Email,
		FullName:     o.Spec.FullName,
		Roles:        o.Spec.Roles,
		PasswordHash: o.Spec.PasswordHash,
	}

	if !o.IsAutoGeneratePassword() && o.Spec.PasswordHash != "" {
		user.PasswordHash = o.Spec.PasswordHash
	}

	if o.Spec.Metadata != nil {
		user.Metadata = o.Spec.Metadata.Data
	}

	return user, nil
}

func (h *userApiClient) Get(o *elasticsearchapicrd.User) (object *olivere.XPackSecurityPutUserRequest, err error) {
	u, err := h.Client().UserGet(o.GetExternalName())
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, nil
	}

	object = &olivere.XPackSecurityPutUserRequest{
		Enabled:  u.Enabled,
		Email:    u.Email,
		FullName: u.Fullname,
		Metadata: u.Metadata,
		Roles:    u.Roles,
		Password: o.Status.PasswordHash,
	}

	return object, nil
}

func (h *userApiClient) Create(object *olivere.XPackSecurityPutUserRequest, o *elasticsearchapicrd.User) (err error) {
	return h.Client().UserCreate(o.GetExternalName(), object)
}

func (h *userApiClient) Update(object *olivere.XPackSecurityPutUserRequest, o *elasticsearchapicrd.User) (err error) {
	return h.Client().UserUpdate(o.GetExternalName(), object, o.IsProtected())
}

func (h *userApiClient) Delete(o *elasticsearchapicrd.User) (err error) {
	return h.Client().UserDelete(o.GetExternalName())
}

func (h *userApiClient) Diff(currentOject *olivere.XPackSecurityPutUserRequest, expectedObject *olivere.XPackSecurityPutUserRequest, originalObject *olivere.XPackSecurityPutUserRequest, o *elasticsearchapicrd.User, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().UserDiff(currentOject, expectedObject, originalObject)
}
