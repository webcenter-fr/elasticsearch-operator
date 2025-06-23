package v1

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (t *TestSuite) TestSetupComponentTemplateWebhook() {
	var (
		o   *ComponentTemplate
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:        "webhook",
			RawTemplate: ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Name:        "webhook",
			RawTemplate: ptr.To(`{}`),
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name:        "webhook",
			RawTemplate: ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Name:        "webhook",
			RawTemplate: ptr.To(`{}`),
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			RawTemplate:      ptr.To(`{}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify any settings
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook6",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when set rawTemplate and settings
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook7",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			RawTemplate: ptr.To(`{}`),
			Settings: &apis.MapAny{
				Data: map[string]any{
					"test": "test",
				},
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when rawTemplate is invalid
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook8",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			RawTemplate: ptr.To(`{"sfdfdf"}`),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
