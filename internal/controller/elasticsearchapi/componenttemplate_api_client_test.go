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

func TestComponentTemplateBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.ComponentTemplate
		ct         *olivere.IndicesGetComponentTemplate
		expectedCt *olivere.IndicesGetComponentTemplate
		err        error
		client     *componentTemplateApiClient
	)

	client = &componentTemplateApiClient{}

	// With minimal parameters
	o = &elasticsearchapicrd.ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
		},
	}

	expectedCt = &olivere.IndicesGetComponentTemplate{
		Template: &olivere.IndicesGetComponentTemplateData{
			Settings: map[string]any{},
			Mappings: map[string]any{},
			Aliases:  map[string]any{},
		},
	}

	ct, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedCt, ct)

	// With all parameters
	o = &elasticsearchapicrd.ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
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
	}

	expectedCt = &olivere.IndicesGetComponentTemplate{
		Template: &olivere.IndicesGetComponentTemplateData{
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

	ct, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedCt, ct)
}
