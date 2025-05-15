package elasticsearchapi

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIndexTemplateBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.IndexTemplate
		it         *olivere.IndicesGetIndexTemplate
		expectedIt *olivere.IndicesGetIndexTemplate
		err        error
	)

	client := &indexTemplateApiClient{}

	// With minimal parameters
	o = &elasticsearchapicrd.IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.IndexTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			IndexPatterns: []string{"*"},
		},
	}

	expectedIt = &olivere.IndicesGetIndexTemplate{
		IndexPatterns: []string{"*"},
	}

	it, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedIt, it)

	// With all parameters
	o = &elasticsearchapicrd.IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.IndexTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			IndexPatterns: []string{"*"},
			ComposedOf:    []string{"component1"},
			Priority:      1,
			Version:       1,
			Template: &elasticsearchapicrd.IndexTemplateData{
				Settings: &apis.MapAny{
					Data: map[string]any{
						"number_of_shards": 1,
					},
				},
				Mappings: &apis.MapAny{
					Data: map[string]any{
						"_source": map[string]any{
							"enabled": false,
						},
					},
				},
				Aliases: &apis.MapAny{
					Data: map[string]any{
						"key": "value",
					},
				},
			},
			Meta: &apis.MapAny{
				Data: map[string]any{
					"key": "value",
				},
			},
			AllowAutoCreate: true,
		},
	}

	expectedIt = &olivere.IndicesGetIndexTemplate{
		IndexPatterns: []string{"*"},
		ComposedOf:    []string{"component1"},
		Priority:      1,
		Version:       1,
		Meta: map[string]any{
			"key": "value",
		},
		AllowAutoCreate: true,
		Template: &olivere.IndicesGetIndexTemplateData{
			Settings: map[string]any{
				"number_of_shards": 1,
			},
			Mappings: map[string]any{
				"_source": map[string]any{
					"enabled": false,
				},
			},
			Aliases: map[string]any{
				"key": "value",
			},
		},
	}

	it, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedIt, it)
}
