package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

type roleApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler]
}

func newRoleApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler] {
	return &roleApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler](client),
	}
}

func (h *roleApiClient) Build(o *elasticsearchapicrd.Role) (role *eshandler.XPackSecurityRole, err error) {

	role = &eshandler.XPackSecurityRole{
		Cluster: o.Spec.Cluster,
		RunAs:   o.Spec.RunAs,
	}

	if o.Spec.Global != "" {
		global := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Global), &global); err != nil {
			return nil, err
		}
		role.Global = global
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, err
		}
		role.Metadata = meta
	}

	if o.Spec.TransientMetadata != "" {
		tm := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.TransientMetadata), &tm); err != nil {
			return nil, err
		}
		role.TransientMetadata = tm
	}

	if o.Spec.Applications != nil {
		role.Applications = make([]eshandler.XPackSecurityApplicationPrivileges, 0, len(o.Spec.Applications))
		for _, application := range o.Spec.Applications {
			role.Applications = append(role.Applications, eshandler.XPackSecurityApplicationPrivileges{
				Application: application.Application,
				Privileges:  application.Privileges,
				Resources:   application.Resources,
			})
		}
	}

	if o.Spec.Indices != nil {
		role.Indices = make([]eshandler.XPackSecurityIndicesPermissions, 0, len(o.Spec.Indices))
		for _, indice := range o.Spec.Indices {
			i := eshandler.XPackSecurityIndicesPermissions{
				Names:                  indice.Names,
				Privileges:             indice.Privileges,
				Query:                  indice.Query,
				AllowRestrictedIndices: indice.AllowRestrictedIndices,
			}
			if indice.FieldSecurity != "" {
				fs := make(map[string]any)
				if err := json.Unmarshal([]byte(indice.FieldSecurity), &fs); err != nil {
					return nil, err
				}
				i.FieldSecurity = fs
			}
			role.Indices = append(role.Indices, i)
		}
	}

	return role, nil
}

func (h *roleApiClient) Get(o *elasticsearchapicrd.Role) (object *eshandler.XPackSecurityRole, err error) {
	return h.Client().RoleGet(o.GetExternalName())
}

func (h *roleApiClient) Create(object *eshandler.XPackSecurityRole, o *elasticsearchapicrd.Role) (err error) {
	return h.Client().RoleUpdate(o.GetExternalName(), object)
}

func (h *roleApiClient) Update(object *eshandler.XPackSecurityRole, o *elasticsearchapicrd.Role) (err error) {
	return h.Client().RoleUpdate(o.GetExternalName(), object)

}

func (h *roleApiClient) Delete(o *elasticsearchapicrd.Role) (err error) {
	return h.Client().RoleDelete(o.GetExternalName())

}

func (h *roleApiClient) Diff(currentOject *eshandler.XPackSecurityRole, expectedObject *eshandler.XPackSecurityRole, originalObject *eshandler.XPackSecurityRole, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().RoleDiff(currentOject, expectedObject, originalObject)
}
