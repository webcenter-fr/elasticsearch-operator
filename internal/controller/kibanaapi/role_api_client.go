package kibanaapi

import (
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
)

type roleApiClient struct {
	*controller.BasicRemoteExternalReconciler[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler]
}

func newRoleApiClient(client kbhandler.KibanaHandler) controller.RemoteExternalReconciler[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler] {
	return &roleApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*kibanaapicrd.Role, *kbapi.KibanaRole, kbhandler.KibanaHandler](client),
	}
}

func (h *roleApiClient) Build(o *kibanaapicrd.Role) (role *kbapi.KibanaRole, err error) {
	role = &kbapi.KibanaRole{
		Name: o.GetExternalName(),
	}

	if o.Spec.TransientMedata != nil {
		role.TransientMedata = &kbapi.KibanaRoleTransientMetadata{
			Enabled: o.Spec.TransientMedata.Enabled,
		}
	}

	if o.Spec.Metadata != nil {
		role.Metadata = o.Spec.Metadata.Data
	}

	if o.Spec.Elasticsearch != nil {
		role.Elasticsearch = &kbapi.KibanaRoleElasticsearch{
			Cluster: o.Spec.Elasticsearch.Cluster,
			RunAs:   o.Spec.Elasticsearch.RunAs,
		}

		if len(o.Spec.Elasticsearch.Indices) > 0 {
			role.Elasticsearch.Indices = make([]kbapi.KibanaRoleElasticsearchIndice, 0, len(o.Spec.Elasticsearch.Indices))
			for _, indice := range o.Spec.Elasticsearch.Indices {
				i := kbapi.KibanaRoleElasticsearchIndice{
					Names:      indice.Names,
					Privileges: indice.Privileges,
				}

				if indice.Query != "" {
					i.Query = indice.Query
				}

				if indice.FieldSecurity != nil {
					i.FieldSecurity = indice.FieldSecurity.Data
				}

				role.Elasticsearch.Indices = append(role.Elasticsearch.Indices, i)
			}
		}
	}

	if len(o.Spec.Kibana) > 0 {
		role.Kibana = make([]kbapi.KibanaRoleKibana, 0, len(o.Spec.Kibana))
		for _, kibana := range o.Spec.Kibana {
			role.Kibana = append(role.Kibana, kbapi.KibanaRoleKibana{
				Base:    kibana.Base,
				Feature: kibana.Feature,
				Spaces:  kibana.Spaces,
			})
		}
	}

	return role, nil
}

func (h *roleApiClient) Get(o *kibanaapicrd.Role) (object *kbapi.KibanaRole, err error) {
	return h.Client().RoleGet(o.GetExternalName())
}

func (h *roleApiClient) Create(object *kbapi.KibanaRole, o *kibanaapicrd.Role) (err error) {
	return h.Client().RoleUpdate(object)
}

func (h *roleApiClient) Update(object *kbapi.KibanaRole, o *kibanaapicrd.Role) (err error) {
	return h.Client().RoleUpdate(object)
}

func (h *roleApiClient) Delete(o *kibanaapicrd.Role) (err error) {
	return h.Client().RoleDelete(o.GetExternalName())
}

func (h *roleApiClient) Diff(currentOject *kbapi.KibanaRole, expectedObject *kbapi.KibanaRole, originalObject *kbapi.KibanaRole, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	patchResult, err = h.Client().RoleDiff(currentOject, expectedObject, originalObject)

	if err == nil && patchResult == nil {
		panic("The kbhandler return nil patchresult and nil error")
	}

	return patchResult, err
}
