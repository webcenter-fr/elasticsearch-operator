package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupRoleMappingWebhook() {
	var (
		o   *RoleMapping
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name: "webhook",
			Roles: []string{
				"test",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name: "webhook",
			Roles: []string{
				"test",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name: "webhook",
			Roles: []string{
				"test",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name: "webhook",
			Roles: []string{
				"test",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			Roles: []string{
				"test",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"username": "*",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
