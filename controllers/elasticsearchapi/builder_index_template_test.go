package elasticsearchapi

import (
	"testing"

	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildIndexTemplate(t *testing.T) {

	var (
		o          *elasticsearchapicrd.IndexTemplate
		it         *olivere.IndicesGetIndexTemplate
		expectedIt *olivere.IndicesGetIndexTemplate
		err        error
	)

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

	it, err = BuildIndexTemplate(o)
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
				Settings: `{
					"number_of_shards": 1
				}`,
				Mappings: `{
					"_source": {
					  "enabled": false
					}
				}`,
				Aliases: `{
					"key": "value"	
				}`,
			},
			Meta: `{
				"key": "value"
			}`,
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
				"number_of_shards": float64(1),
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

	it, err = BuildIndexTemplate(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedIt, it)

}
