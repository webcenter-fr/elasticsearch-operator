package elasticsearchapi

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestIndexLifecyclePolicyBuild(t *testing.T) {
	var (
		o           *elasticsearchapicrd.IndexLifecyclePolicy
		ilm         *olivere.XPackIlmGetLifecycleResponse
		expectedIlm *olivere.XPackIlmGetLifecycleResponse
		err         error
		client      *indexLifecyclePolicyApiClient
	)

	client = &indexLifecyclePolicyApiClient{}

	// With minimal parameters
	o = &elasticsearchapicrd.IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
		},
	}

	_, err = client.Build(o)
	assert.NoError(t, err)

	// With all parameters
	o = &elasticsearchapicrd.IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Policy: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicy{
				Phases: elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhases{
					Warm: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhasesPhase{
						MinAge: ptr.To("10d"),
						Actions: apis.MapAny{
							Data: map[string]any{
								"forcemerge": map[string]any{
									"max_num_segments": 1,
								},
							},
						},
					},
				},
			},
		},
	}

	expectedIlm = &olivere.XPackIlmGetLifecycleResponse{
		Policy: map[string]any{
			"phases": map[string]any{
				"warm": map[string]any{
					"min_age": "10d",
					"actions": map[string]any{
						"forcemerge": map[string]any{
							"max_num_segments": float64(1),
						},
					},
				},
			},
		},
	}

	ilm, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedIlm, ilm)
}
