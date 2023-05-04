package elasticsearchapi

import (
	"encoding/json"

	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

// BuildUser permit to convert to User object
func BuildUser(o *elasticsearchapicrd.User) (user *olivere.XPackSecurityPutUserRequest, err error) {
	user = &olivere.XPackSecurityPutUserRequest{
		Enabled:      o.Spec.Enabled,
		Email:        o.Spec.Email,
		FullName:     o.Spec.FullName,
		Roles:        o.Spec.Roles,
		PasswordHash: o.Spec.PasswordHash,
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, err
		}
		user.Metadata = meta
	}

	return user, nil
}
