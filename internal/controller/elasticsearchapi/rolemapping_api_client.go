package elasticsearchapi

import (
	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type roleMappingApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *esapi.SecurityRoleMapping, eshandler.ElasticsearchHandler]
}

func newRoleMappingApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *esapi.SecurityRoleMapping, eshandler.ElasticsearchHandler] {
	return &roleMappingApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *esapi.SecurityRoleMapping, eshandler.ElasticsearchHandler](client),
	}
}

func (h *roleMappingApiClient) Build(o *elasticsearchapicrd.RoleMapping) (rm *esapi.SecurityRoleMapping, err error) {
	rm = &esapi.SecurityRoleMapping{
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

func (h *roleMappingApiClient) Get(o *elasticsearchapicrd.RoleMapping) (object *esapi.SecurityRoleMapping, err error) {
	return h.Client().RoleMappingGet(o.GetExternalName())
}

func (h *roleMappingApiClient) Create(object *esapi.SecurityRoleMapping, o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingUpdate(o.GetExternalName(), object)
}

func (h *roleMappingApiClient) Update(object *esapi.SecurityRoleMapping, o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingUpdate(o.GetExternalName(), object)
}

func (h *roleMappingApiClient) Delete(o *elasticsearchapicrd.RoleMapping) (err error) {
	return h.Client().RoleMappingDelete(o.GetExternalName())
}

func (h *roleMappingApiClient) Diff(currentOject *esapi.SecurityRoleMapping, expectedObject *esapi.SecurityRoleMapping, originalObject *esapi.SecurityRoleMapping, o *elasticsearchapicrd.RoleMapping, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().RoleMappingDiff(currentOject, expectedObject, originalObject)
}
