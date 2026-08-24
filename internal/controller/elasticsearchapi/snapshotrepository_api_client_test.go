package elasticsearchapi

import (
	"testing"

	esapi "github.com/disaster37/elasticsearch/v9/api"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSnapshotRepositoryBuild(t *testing.T) {
	var (
		o          *elasticsearchapicrd.SnapshotRepository
		sr         *esapi.SnapshotRepository
		expectedSr *esapi.SnapshotRepository
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

	expectedSr = &esapi.SnapshotRepository{}

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
			Settings: &apis.MapAny{
				Data: map[string]any{
					"location": "/snapshot",
				},
			},
		},
	}

	expectedSr = &esapi.SnapshotRepository{
		Type: "fs",
		Settings: map[string]string{
			"location": "/snapshot",
		},
	}

	sr, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSr, sr)
}
