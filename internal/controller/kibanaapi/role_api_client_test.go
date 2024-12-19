package kibanaapi

import (
	"testing"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleBuild(t *testing.T) {
	var (
		o            *kibanaapicrd.Role
		role         *kbapi.KibanaRole
		expectedRole *kbapi.KibanaRole
		err          error
		client       *roleApiClient
	)

	client = &roleApiClient{}

	// Normal case
	o = &kibanaapicrd.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.RoleSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Elasticsearch: &kibanaapicrd.KibanaRoleElasticsearch{
				Cluster: []string{
					"monitor",
				},
				Indices: []kibanaapicrd.KibanaRoleElasticsearchIndice{
					{
						Names: []string{
							"*",
						},
						Privileges: []string{
							"view_index_metadata",
							"monitor",
						},
					},
				},
			},
			Kibana: []kibanaapicrd.KibanaRoleKibana{
				{
					Base: []string{
						"all",
					},
					Spaces: []string{
						"my-space",
					},
				},
			},
		},
	}

	expectedRole = &kbapi.KibanaRole{
		Name: "test",
		Elasticsearch: &kbapi.KibanaRoleElasticsearch{
			Cluster: []string{
				"monitor",
			},
			Indices: []kbapi.KibanaRoleElasticsearchIndice{
				{
					Names: []string{
						"*",
					},
					Privileges: []string{
						"view_index_metadata",
						"monitor",
					},
				},
			},
		},
		Kibana: []kbapi.KibanaRoleKibana{
			{
				Base: []string{
					"all",
				},
				Spaces: []string{
					"my-space",
				},
			},
		},
	}

	role, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)

	// With all parameters
	o = &kibanaapicrd.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.RoleSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Elasticsearch: &kibanaapicrd.KibanaRoleElasticsearch{
				Cluster: []string{
					"monitor",
				},
				RunAs: []string{
					"other_user",
				},
				Indices: []kibanaapicrd.KibanaRoleElasticsearchIndice{
					{
						Names: []string{
							"*",
						},
						Privileges: []string{
							"view_index_metadata",
							"monitor",
						},
						FieldSecurity: `
	{
		"grant" : [ "title", "body" ]
	}
						`,
						Query: `
	{
		"match": {
			"title": "foo"
		}
	}`,
					},
				},
			},
			Metadata: `
{
	"version" : 1
}
			`,
			TransientMedata: &kibanaapicrd.KibanaRoleTransientMetadata{
				Enabled: true,
			},
			Kibana: []kibanaapicrd.KibanaRoleKibana{
				{
					Base: []string{
						"all",
					},
					Spaces: []string{
						"my-space",
					},
					Feature: map[string][]string{
						"discover": {
							"all",
						},
					},
				},
			},
		},
	}

	expectedRole = &kbapi.KibanaRole{
		Name: "test",
		Metadata: map[string]any{
			"version": float64(1),
		},
		TransientMedata: &kbapi.KibanaRoleTransientMetadata{
			Enabled: true,
		},
		Elasticsearch: &kbapi.KibanaRoleElasticsearch{
			Cluster: []string{
				"monitor",
			},
			RunAs: []string{
				"other_user",
			},
			Indices: []kbapi.KibanaRoleElasticsearchIndice{
				{
					Names: []string{
						"*",
					},
					Privileges: []string{
						"view_index_metadata",
						"monitor",
					},
					FieldSecurity: map[string]any{
						"grant": []any{
							"title",
							"body",
						},
					},
					Query: `
	{
		"match": {
			"title": "foo"
		}
	}`,
				},
			},
		},
		Kibana: []kbapi.KibanaRoleKibana{
			{
				Base: []string{
					"all",
				},
				Spaces: []string{
					"my-space",
				},
				Feature: map[string][]string{
					"discover": {
						"all",
					},
				},
			},
		},
	}

	role, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
}
