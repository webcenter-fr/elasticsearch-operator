package kibanaapi

import (
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
)

// BuildUserSpace permit to build user space
func BuildUserSpace(o *kibanaapicrd.UserSpace) (space *kbapi.KibanaSpace, err error) {

	space = &kbapi.KibanaSpace{
		ID:               o.GetUserSpaceID(),
		Name:             o.Spec.Name,
		Description:      o.Spec.Description,
		DisabledFeatures: o.Spec.DisabledFeatures,
		Initials:         o.Spec.Initials,
		Color:            o.Spec.Color,
	}

	return space, nil
}
