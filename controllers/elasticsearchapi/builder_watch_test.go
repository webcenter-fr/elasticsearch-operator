package elasticsearchapi

import (
	"testing"

	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildWatch(t *testing.T) {

	var (
		o             *elasticsearchapicrd.Watch
		watch         *olivere.XPackWatch
		expectedWatch *olivere.XPackWatch
		err           error
	)

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
			Trigger: `{
				"schedule" : { "cron" : "0 0/1 * * * ?" }
			}`,
			Input: `{
				"search": {
					"key": "value"
				}	
			}`,
			Condition: `{
				"compare" : { "ctx.payload.hits.total" : { "gt" : 0 }}
			}`,
			Actions: `{
				"email_admin" : {
				  "email" : {
					"to" : "admin@domain.host.com",
					"subject" : "404 recently encountered"
				  }
				}
			}`,
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

	watch, err = BuildWatch(o)
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
			Trigger: `{
				"schedule" : { "cron" : "0 0/1 * * * ?" }
			}`,
			Input: `{
				"search": {
					"key": "value"
				}	
			}`,
			Condition: `{
				"compare" : { "ctx.payload.hits.total" : { "gt" : 0 }}
			}`,
			Actions: `{
				"email_admin" : {
				  "email" : {
					"to" : "admin@domain.host.com",
					"subject" : "404 recently encountered"
				  }
				}
			}`,
			Transform: `{
				"key3": "value3"	
			}`,
			ThrottlePeriod:         "1d",
			ThrottlePeriodInMillis: 10,
			Metadata: `{
				"key2": "value2"
			}`,
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

	watch, err = BuildWatch(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedWatch, watch)

}
