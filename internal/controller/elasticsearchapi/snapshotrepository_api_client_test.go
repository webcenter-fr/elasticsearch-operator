package elasticsearchapi

import (
	"testing"

	olivere "github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSnapshotRepositoryBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.SnapshotRepository
		sr         *olivere.SnapshotRepositoryMetaData
		expectedSr *olivere.SnapshotRepositoryMetaData
		err        error
		client     *snapshotRepositoryApiClient
	)

	client = &snapshotRepositoryApiClient{}

	// With minimal parameters
	o = &elasticsearchapicrd.SnapshotRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.SnapshotRepositorySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
		},
	}

	expectedSr = &olivere.SnapshotRepositoryMetaData{
		Settings: map[string]any{},
	}

	sr, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSr, sr)

	// With all parameters
	o = &elasticsearchapicrd.SnapshotRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.SnapshotRepositorySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Type: "fs",
			Settings: `
{
	"location": "/snapshot"
}
			`,
		},
	}

	expectedSr = &olivere.SnapshotRepositoryMetaData{
		Type: "fs",
		Settings: map[string]any{
			"location": "/snapshot",
		},
	}

	sr, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSr, sr)
}
