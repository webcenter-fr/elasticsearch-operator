package elasticsearchapi

import (
	"testing"

	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildUser(t *testing.T) {
	var (
		o            *elasticsearchapicrd.User
		user         *olivere.XPackSecurityPutUserRequest
		expectedUser *olivere.XPackSecurityPutUserRequest
		err          error
	)

	// When no metadata
	o = &elasticsearchapicrd.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.UserSpec{
			Enabled:      true,
			Email:        "test@no.no",
			FullName:     "test",
			PasswordHash: "hash",
			Roles: []string{
				"admin",
			},
		},
	}

	expectedUser = &olivere.XPackSecurityPutUserRequest{
		Enabled:  true,
		Email:    "test@no.no",
		FullName: "test",
		Roles: []string{
			"admin",
		},
		PasswordHash: "hash",
	}

	user, err = BuildUser(o)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	// When metadata
	o = &elasticsearchapicrd.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.UserSpec{
			Enabled:      true,
			Email:        "test@no.no",
			FullName:     "test",
			PasswordHash: "hash",
			Roles: []string{
				"admin",
			},
			Metadata: `{"foo": "bar"}`,
		},
	}

	expectedUser = &olivere.XPackSecurityPutUserRequest{
		Enabled:  true,
		Email:    "test@no.no",
		FullName: "test",
		Roles: []string{
			"admin",
		},
		PasswordHash: "hash",
		Metadata: map[string]interface{}{
			"foo": "bar",
		},
	}

	user, err = BuildUser(o)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}
