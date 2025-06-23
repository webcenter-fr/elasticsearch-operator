package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupLogstashPipelineWebhook() {
	var (
		o   *LogstashPipeline
		err error
	)

	// Need failed when create same resource by external name on same managed cluster
	// Check we can update it
	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Name:     "webhook",
			Pipeline: "test",
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Name:     "webhook",
			Pipeline: "test",
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create same resource by external name on same external cluster
	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ExternalKibanaRef: &shared.KibanaExternalRef{
					Address: "https://test.local",
				},
			},
			Name:     "webhook",
			Pipeline: "test",
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ExternalKibanaRef: &shared.KibanaExternalRef{
					Address: "https://test.local",
				},
			},
			Name:     "webhook",
			Pipeline: "test",
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Kibana
	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{},
			Pipeline:  "test",
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
