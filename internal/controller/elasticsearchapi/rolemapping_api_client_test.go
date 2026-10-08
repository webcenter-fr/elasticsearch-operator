package elasticsearchapi

import (
	"testing"

	esapi "github.com/disaster37/elasticsearch/v9/api"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleMappingBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.RoleMapping
		rm         *esapi.SecurityRoleMapping
		expectedRm *esapi.SecurityRoleMapping
		err        error
	)

	client := &roleMappingApiClient{}

	// With minimal info
	o = &elasticsearchapicrd.RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Enabled: true,
			Roles: []string{
				"superuser",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"any": []map[string]any{
						{
							"field": map[string]any{
								"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
							},
						},
						{
							"field": map[string]any{
								"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD",
							},
						},
					},
				},
			},
		},
	}

	expectedRm = &esapi.SecurityRoleMapping{
		Enabled: true,
		Roles: []string{
			"superuser",
		},
		Rules: map[string]any{
			"any": []map[string]any{
				{
					"field": map[string]any{
						"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
					},
				},
				{
					"field": map[string]any{
						"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD",
					},
				},
			},
		},
		Metadata: map[string]any{},
	}

	rm, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedRm, rm)

	// With metadata
	o = &elasticsearchapicrd.RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.RoleMappingSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Enabled: true,
			Roles: []string{
				"superuser",
			},
			Rules: &apis.MapAny{
				Data: map[string]any{
					"any": []map[string]any{
						{
							"field": map[string]any{
								"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
							},
						},
						{
							"field": map[string]any{
								"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD",
							},
						},
					},
				},
			},
			Metadata: &apis.MapAny{
				Data: map[string]any{
					"meta1": "data1",
				},
			},
		},
	}

	expectedRm = &esapi.SecurityRoleMapping{
		Enabled: true,
		Roles: []string{
			"superuser",
		},
		Rules: map[string]any{
			"any": []map[string]any{
				{
					"field": map[string]any{
						"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
					},
				},
				{
					"field": map[string]any{
						"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD",
					},
				},
			},
		},
		Metadata: map[string]any{
			"meta1": "data1",
		},
	}

	rm, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedRm, rm)
}
