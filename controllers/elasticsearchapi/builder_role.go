package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
)

// BuildRole permit to build role
func BuildRole(o *elasticsearchapicrd.Role) (role *eshandler.XPackSecurityRole, err error) {

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
				Names:      indice.Names,
				Privileges: indice.Privileges,
				Query:      indice.Query,
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
