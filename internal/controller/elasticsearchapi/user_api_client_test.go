package elasticsearchapi

import (
	"testing"

	eshandler "github.com/disaster37/es-handler/v9"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestUserBuild(t *testing.T) {
	var (
		o            *elasticsearchapicrd.User
		user         *eshandler.SecurityPutUserRequest
		expectedUser *eshandler.SecurityPutUserRequest
		err          error
	)

	client := &userApiClient{}

	// When no metadata
	o = &elasticsearchapicrd.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.UserSpec{
			Enabled:      ptr.To(true),
			Email:        "test@no.no",
			FullName:     "test",
			PasswordHash: "hash",
			Roles: []string{
				"admin",
			},
		},
	}

	expectedUser = &eshandler.SecurityPutUserRequest{
		Enabled:  true,
		Email:    "test@no.no",
		FullName: "test",
		Roles: []string{
			"admin",
		},
		PasswordHash: "hash",
	}

	user, err = client.Build(o)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)

	// When metadata
	o = &elasticsearchapicrd.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.UserSpec{
			Enabled:      ptr.To(true),
			Email:        "test@no.no",
			FullName:     "test",
			PasswordHash: "hash",
			Roles: []string{
				"admin",
			},
			Metadata: &apis.MapAny{
				Data: map[string]any{
					"foo": "bar",
				},
			},
		},
	}

	expectedUser = &eshandler.SecurityPutUserRequest{
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

	user, err = client.Build(o)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}
