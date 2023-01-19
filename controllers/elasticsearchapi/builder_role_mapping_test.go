package elasticsearchapi

import (
	"testing"

	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildRoleMapping(t *testing.T) {

	var (
		o          *elasticsearchapicrd.RoleMapping
		rm         *olivere.XPackSecurityRoleMapping
		expectedRm *olivere.XPackSecurityRoleMapping
		err        error
	)

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
			Rules: `
{
	"any": [
		{
			"field": {
				"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD"
			}
		},
		{
			"field": {
				"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD"
			}
		}
	]
}
			`,
		},
	}

	expectedRm = &olivere.XPackSecurityRoleMapping{
		Enabled: true,
		Roles: []string{
			"superuser",
		},
		Rules: map[string]any{
			"any": []any{
				map[string]any{
					"field": map[string]any{
						"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
					},
				},
				map[string]any{
					"field": map[string]any{
						"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD",
					},
				},
			},
		},
		Metadata: map[string]any{},
	}

	rm, err = BuildRoleMapping(o)
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
			Rules: `
{
	"any": [
		{
			"field": {
				"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD"
			}
		},
		{
			"field": {
				"groups": "CN=ELS_OPS,OU=LOCAL,OU=AD"
			}
		}
	]
}
			`,
			Metadata: `
{
	"meta1": "data1"
}
			`,
		},
	}

	expectedRm = &olivere.XPackSecurityRoleMapping{
		Enabled: true,
		Roles: []string{
			"superuser",
		},
		Rules: map[string]any{
			"any": []any{
				map[string]any{
					"field": map[string]any{
						"groups": "CN=ELS_ADMINS,OU=LOCAL,OU=AD",
					},
				},
				map[string]any{
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

	rm, err = BuildRoleMapping(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedRm, rm)

}
