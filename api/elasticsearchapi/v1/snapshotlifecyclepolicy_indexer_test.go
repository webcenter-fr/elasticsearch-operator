package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupSnapshotLifecyclePolicyIndexer() {
	// Add roleMapping to force indexer execution

	slm := &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Schedule:   "test",
			Name:       "policy-name",
			Repository: "snapshot",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), slm)
	assert.NoError(t.T(), err)
}
