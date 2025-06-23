package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupWatchWebhook() {
	var (
		o   *Watch
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name: "webhook",
			Trigger: &apis.MapAny{
				Data: map[string]any{},
			},
			Input: &apis.MapAny{
				Data: map[string]any{},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name: "webhook",
			Trigger: &apis.MapAny{
				Data: map[string]any{},
			},
			Input: &apis.MapAny{
				Data: map[string]any{},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name: "webhook",
			Trigger: &apis.MapAny{
				Data: map[string]any{},
			},
			Input: &apis.MapAny{
				Data: map[string]any{},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name: "webhook",
			Trigger: &apis.MapAny{
				Data: map[string]any{},
			},
			Input: &apis.MapAny{
				Data: map[string]any{},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			Trigger: &apis.MapAny{
				Data: map[string]any{},
			},
			Input: &apis.MapAny{
				Data: map[string]any{},
			},
			Condition: &apis.MapAny{
				Data: map[string]any{},
			},
			Actions: &apis.MapAny{
				Data: map[string]any{},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
