package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupSnapshotLifecyclePolicyWebhook() {
	var (
		o   *SnapshotLifecyclePolicy
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			SnapshotLifecyclePolicyName: "webhook",
			Schedule:                    "test",
			Repository:                  "snapshot",
			Name:                        "policy-name",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			SnapshotLifecyclePolicyName: "webhook",
			Schedule:                    "test",
			Repository:                  "snapshot",
			Name:                        "policy-name",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			SnapshotLifecyclePolicyName: "webhook",
			Schedule:                    "test",
			Repository:                  "snapshot",
			Name:                        "policy-name",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			SnapshotLifecyclePolicyName: "webhook",
			Schedule:                    "test",
			Repository:                  "snapshot",
			Name:                        "policy-name",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: SnapshotLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			Schedule:         "test",
			Repository:       "snapshot",
			Name:             "policy-name",
			Config: SLMConfig{
				Indices: []string{
					"test",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
