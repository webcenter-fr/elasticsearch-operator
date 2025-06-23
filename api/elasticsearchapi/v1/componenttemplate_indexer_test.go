package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (t *TestSuite) TestSetupComponentTemplateIndexer() {
	// Add ComponentTemplate to force  indexer execution

	o := &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ComponentTemplateSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			RawTemplate: ptr.To(`{}`),
		},
	}

	err := t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
}
