package elasticsearchapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetUserSecretWhenAutoGeneratePassword(t *testing.T) {
	u := &elasticsearchapicrd.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.UserSpec{},
	}

	assert.Equal(t, "test-credential-es", GetUserSecretWhenAutoGeneratePassword(u))

}
