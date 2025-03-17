package elasticsearchapi

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWatchBuild(t *testing.T) {
	var (
		o             *elasticsearchapicrd.Watch
		watch         *olivere.XPackWatch
		expectedWatch *olivere.XPackWatch
		err           error
	)

	client := &watchApiClient{}

	// With minimal parameters
	o = &elasticsearchapicrd.Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Trigger: &apis.MapAny{
				Data: map[string]any{
					"schedule": map[string]any{
						"cron": "0 0/1 * * * ?",
					},
				},
			},
			Input: &apis.MapAny{
				Data: map[string]any{
					"search": map[string]any{
						"key": "value",
					},
				},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{
					"compare": map[string]any{
						"ctx.payload.hits.total": map[string]any{
							"gt": 0,
						},
					},
				},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{
					"email_admin": map[string]any{
						"email": map[string]any{
							"to":      "admin@domain.host.com",
							"subject": "404 recently encountered",
						},
					},
				},
			},
		},
	}

	expectedWatch = &olivere.XPackWatch{
		Trigger: map[string]map[string]any{
			"schedule": {
				"cron": "0 0/1 * * * ?",
			},
		},
		Input: map[string]map[string]any{
			"search": {
				"key": "value",
			},
		},
		Condition: map[string]map[string]any{
			"compare": {
				"ctx.payload.hits.total": map[string]any{
					"gt": float64(0),
				},
			},
		},
		Actions: map[string]map[string]any{
			"email_admin": {
				"email": map[string]any{
					"to":      "admin@domain.host.com",
					"subject": "404 recently encountered",
				},
			},
		},
	}

	watch, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, watch)

	// With all parameters
	o = &elasticsearchapicrd.Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Trigger: &apis.MapAny{
				Data: map[string]any{
					"schedule": map[string]any{
						"cron": "0 0/1 * * * ?",
					},
				},
			},
			Input: &apis.MapAny{
				Data: map[string]any{
					"search": map[string]any{
						"key": "value",
					},
				},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{
					"compare": map[string]any{
						"ctx.payload.hits.total": map[string]any{
							"gt": 0,
						},
					},
				},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{
					"email_admin": map[string]any{
						"email": map[string]any{
							"to":      "admin@domain.host.com",
							"subject": "404 recently encountered",
						},
					},
				},
			},
			Transform: &apis.MapAny{
				Data: map[string]any{
					"key3": "value3",
				},
			},
			ThrottlePeriod:         "1d",
			ThrottlePeriodInMillis: 10,
			Metadata: &apis.MapAny{
				Data: map[string]any{
					"key2": "value2",
				},
			},
		},
	}

	expectedWatch = &olivere.XPackWatch{
		Trigger: map[string]map[string]any{
			"schedule": {
				"cron": "0 0/1 * * * ?",
			},
		},
		Input: map[string]map[string]any{
			"search": {
				"key": "value",
			},
		},
		Condition: map[string]map[string]any{
			"compare": {
				"ctx.payload.hits.total": map[string]any{
					"gt": float64(0),
				},
			},
		},
		Actions: map[string]map[string]any{
			"email_admin": {
				"email": map[string]any{
					"to":      "admin@domain.host.com",
					"subject": "404 recently encountered",
				},
			},
		},
		Transform: map[string]any{
			"key3": "value3",
		},
		ThrottlePeriod:         "1d",
		ThrottlePeriodInMillis: 10,
		Metadata: map[string]any{
			"key2": "value2",
		},
	}

	watch, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, watch)
}
