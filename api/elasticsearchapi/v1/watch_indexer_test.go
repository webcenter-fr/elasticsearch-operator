package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupWatchIndexer() {
	// Add user to force  indexer execution

	watch := &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: WatchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
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

	err := t.k8sClient.Create(context.Background(), watch)
	assert.NoError(t.T(), err)
}
