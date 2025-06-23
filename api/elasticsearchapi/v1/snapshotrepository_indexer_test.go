package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupSnapshotRepositoryIndexer() {
	// Add SnapshotRepository to force indexer execution

	snapshotRepository := &SnapshotRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: SnapshotRepositorySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Type: "local",
			Settings: &apis.MapAny{
				Data: map[string]any{
					"path": "/mnt/snpshot",
				},
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), snapshotRepository)
	assert.NoError(t.T(), err)
}
