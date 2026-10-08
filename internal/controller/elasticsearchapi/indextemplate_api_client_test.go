package elasticsearchapi

import (
	"testing"

	eshandlerpatch "github.com/disaster37/es-handler/v9/patch"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIndexTemplateBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.IndexTemplate
		it         *eshandlerpatch.IndexTemplate
		expectedIt *eshandlerpatch.IndexTemplate
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

	expectedIt = &eshandlerpatch.IndexTemplate{
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

	expectedIt = &eshandlerpatch.IndexTemplate{
		IndexPatterns: []string{"*"},
		ComposedOf:    []string{"component1"},
		Priority:      1,
		Version:       1,
		Meta: map[string]any{
			"key": "value",
		},
		Template: &eshandlerpatch.IndexTemplateData{
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
