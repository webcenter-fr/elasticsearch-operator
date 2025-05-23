package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type roleMappingApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler]
}

func newRoleMappingApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler] {
	return &roleMappingApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler](client),
	}
}

func (h *roleMappingApiClient) Build(o *elasticsearchapicrd.RoleMapping) (rm *olivere.XPackSecurityRoleMapping, err error) {
	rm = &olivere.XPackSecurityRoleMapping{
		Enabled:  o.Spec.Enabled,
		Roles:    o.Spec.Roles,
		Metadata: make(map[string]any), // Fix issue on V8, metadata can't be null
	}

	if o.Spec.Rules != nil {
		rm.Rules = o.Spec.Rules.Data
	}

	if o.Spec.Metadata != nil {
		rm.Metadata = o.Spec.Metadata.Data
	}

	return rm, nil
}

func (h *roleMappingApiClient) Get(o *elasticsearchapicrd.RoleMapping) (object *olivere.XPackSecurityRoleMapping, err error) {
	return h.Client().RoleMappingGet(o.GetExternalName())
}

func (h *roleMappingApiClient) Create(object *olivere.XPackSecurityRoleMapping, o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingUpdate(o.GetExternalName(), object)
}

func (h *roleMappingApiClient) Update(object *olivere.XPackSecurityRoleMapping, o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingUpdate(o.GetExternalName(), object)
}

func (h *roleMappingApiClient) Delete(o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingDelete(o.GetExternalName())
}

func (h *roleMappingApiClient) Diff(currentOject *olivere.XPackSecurityRoleMapping, expectedObject *olivere.XPackSecurityRoleMapping, originalObject *olivere.XPackSecurityRoleMapping, o *elasticsearchapicrd.RoleMapping, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().RoleMappingDiff(currentOject, expectedObject, originalObject)
}
