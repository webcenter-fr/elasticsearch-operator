package helper

import (
	"testing"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

func TestSetLastOriginal(t *testing.T) {
	var (
		role *elasticsearchapicrd.Role
		o    *eshandler.XPackSecurityRole
		o2   *eshandler.XPackSecurityRole
		err  error
	)

	// Normale use case
	role = &elasticsearchapicrd.Role{
		Status: elasticsearchapicrd.RoleStatus{},
	}

	o = &eshandler.XPackSecurityRole{
		Cluster: []string{
			"test",
		},
	}

	err = SetLastOriginal(role, o)

	assert.NoError(t, err)
	assert.NotEmpty(t, role.Status.OriginalObject)

	o2 = &eshandler.XPackSecurityRole{}
	err = UnZipBase64Decode(role.Status.OriginalObject, o2)
	assert.NoError(t, err)
	assert.Equal(t, o, o2)

	// When unzip empty string
	o2 = &eshandler.XPackSecurityRole{}
	err = UnZipBase64Decode("", o2)
	assert.NoError(t, err)

}
