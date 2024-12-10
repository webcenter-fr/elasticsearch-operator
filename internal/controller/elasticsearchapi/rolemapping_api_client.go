package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

type roleMappingApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler]
}

func newRoleMappingApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler] {
	return &roleMappingApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler](client),
	}
}

func (h *roleMappingApiClient) Build(o *elasticsearchapicrd.RoleMapping) (rm *olivere.XPackSecurityRoleMapping, err error) {
	rm = &olivere.XPackSecurityRoleMapping{
		Enabled:  o.Spec.Enabled,
		Roles:    o.Spec.Roles,
		Metadata: make(map[string]any), // Fix issue on V8, metadata can't be null
	}

	if o.Spec.Rules != "" {
		rules := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Rules), &rules); err != nil {
			return nil, err
		}
		rm.Rules = rules
	}

	if o.Spec.Metadata != "" {
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &rm.Metadata); err != nil {
			return nil, err
		}
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

func (h *roleMappingApiClient) Diff(currentOject *olivere.XPackSecurityRoleMapping, expectedObject *olivere.XPackSecurityRoleMapping, originalObject *olivere.XPackSecurityRoleMapping, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().RoleMappingDiff(currentOject, expectedObject, originalObject)
}
