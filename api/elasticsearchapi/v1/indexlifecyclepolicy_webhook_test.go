package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (t *TestSuite) TestSetupIndexLifecyclePolicyWebhook() {
	var (
		o   *IndexLifecyclePolicy
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:      "webhook",
			RawPolicy: ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:      "webhook",
			RawPolicy: ptr.To(`{}`),
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name:      "webhook",
			RawPolicy: ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name:      "webhook",
			RawPolicy: ptr.To(`{}`),
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			RawPolicy:        ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify any settings
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook6",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when set rawPolicy and policy
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook7",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			RawPolicy: ptr.To(`{}`),
			Policy:    &IndexLifecyclePolicySpecPolicy{},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when rawPolicy is invalid
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook8",
			Namespace: "default",
		},
		Spec: IndexLifecyclePolicySpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			RawPolicy: ptr.To(`{"sfdfdf"}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
