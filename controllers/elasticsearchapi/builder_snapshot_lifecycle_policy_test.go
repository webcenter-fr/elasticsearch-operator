package elasticsearchapi

import (
	"testing"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildSnapshotLifecyclePolicy(t *testing.T) {

	var (
		o           *elasticsearchapicrd.SnapshotLifecyclePolicy
		slm         *eshandler.SnapshotLifecyclePolicySpec
		expectedSLM *eshandler.SnapshotLifecyclePolicySpec
		err         error
	)

	// Normal case
	o = &elasticsearchapicrd.SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:       "<daily-snap-{now/d}>",
			Schedule:   "0 30 1 * * ?",
			Repository: "my_repository",
			Config: elasticsearchapicrd.SLMConfig{
				Indices: []string{
					"data-*",
					"important",
				},
				IgnoreUnavailable:  false,
				IncludeGlobalState: false,
			},
		},
	}

	expectedSLM = &eshandler.SnapshotLifecyclePolicySpec{
		Name:       "<daily-snap-{now/d}>",
		Schedule:   "0 30 1 * * ?",
		Repository: "my_repository",
		Config: eshandler.ElasticsearchSLMConfig{
			Indices: []string{
				"data-*",
				"important",
			},
			IgnoreUnavailable:  false,
			IncludeGlobalState: false,
		},
	}

	slm, err = BuildSnapshotLifecyclePolicy(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSLM, slm)

	// When retention is set
	o = &elasticsearchapicrd.SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchapicrd.SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:       "<daily-snap-{now/d}>",
			Schedule:   "0 30 1 * * ?",
			Repository: "my_repository",
			Config: elasticsearchapicrd.SLMConfig{
				Indices: []string{
					"data-*",
					"important",
				},
				IgnoreUnavailable:  false,
				IncludeGlobalState: false,
			},
			Retention: &elasticsearchapicrd.SLMRetention{
				ExpireAfter: "30d",
				MinCount:    5,
				MaxCount:    50,
			},
		},
	}

	expectedSLM = &eshandler.SnapshotLifecyclePolicySpec{
		Name:       "<daily-snap-{now/d}>",
		Schedule:   "0 30 1 * * ?",
		Repository: "my_repository",
		Config: eshandler.ElasticsearchSLMConfig{
			Indices: []string{
				"data-*",
				"important",
			},
			IgnoreUnavailable:  false,
			IncludeGlobalState: false,
		},
		Retention: &eshandler.ElasticsearchSLMRetention{
			ExpireAfter: "30d",
			MinCount:    5,
			MaxCount:    50,
		},
	}

	slm, err = BuildSnapshotLifecyclePolicy(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSLM, slm)

}
