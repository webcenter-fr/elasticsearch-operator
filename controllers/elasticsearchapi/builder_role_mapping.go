package elasticsearchapi

import (
	"encoding/json"

	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
)

// BuildRoleMapping permit to build roleMapping
func BuildRoleMapping(o *elasticsearchapicrd.RoleMapping) (rm *olivere.XPackSecurityRoleMapping, err error) {

	rm = &olivere.XPackSecurityRoleMapping{
		Enabled: o.Spec.Enabled,
		Roles:   o.Spec.Roles,
	}

	if o.Spec.Rules != "" {
		rules := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Rules), &rules); err != nil {
			return nil, err
		}
		rm.Rules = rules
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, err
		}
		rm.Metadata = meta
	}

	return rm, nil
}
