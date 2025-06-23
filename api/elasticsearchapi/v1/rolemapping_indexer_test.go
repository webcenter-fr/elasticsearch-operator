package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupRoleMappingIndexer() {
	// Add roleMapping to force indexer execution

	roleMapping := &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Roles: []string{
				"superuser",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), roleMapping)
	assert.NoError(t.T(), err)
}
