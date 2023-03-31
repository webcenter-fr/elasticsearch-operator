package kibanaapi

import (
	"encoding/json"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1alpha1"
)

// BuildRole permit to build role
func BuildRole(o *kibanaapicrd.Role) (role *kbapi.KibanaRole, err error) {

	role = &kbapi.KibanaRole{
		Name: o.GetRoleName(),
	}

	if o.Spec.TransientMedata != nil {
		role.TransientMedata = &kbapi.KibanaRoleTransientMetadata{
			Enabled: o.Spec.TransientMedata.Enabled,
		}
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, err
		}
		role.Metadata = meta
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

				if indice.FieldSecurity != "" {
					fs := make(map[string]any)
					if err := json.Unmarshal([]byte(indice.FieldSecurity), &fs); err != nil {
						return nil, err
					}
					i.FieldSecurity = fs
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
