package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupLogstashPipelineIndexer() {
	// Add LogstashPipeline to force  indexer execution

	o := &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Pipeline: "test",
		},
	}

	err := t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
}
