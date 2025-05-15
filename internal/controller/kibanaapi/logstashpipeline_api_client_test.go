package kibanaapi

import (
	"testing"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogstashPipelineBuild(t *testing.T) {
	var (
		o                *kibanaapicrd.LogstashPipeline
		pipeline         *kbapi.LogstashPipeline
		expectedPipeline *kbapi.LogstashPipeline
		err              error
		client           *logstashPipelineApiClient
	)

	client = &logstashPipelineApiClient{}

	// Normal case
	o = &kibanaapicrd.LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Pipeline: "input { stdin {} } output { stdout {} }",
		},
	}

	expectedPipeline = &kbapi.LogstashPipeline{
		ID:       "test",
		Pipeline: "input { stdin {} } output { stdout {} }",
	}

	pipeline, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedPipeline, pipeline)

	// With all parameters
	o = &kibanaapicrd.LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.LogstashPipelineSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Pipeline:    "input { stdin {} } output { stdout {} }",
			Description: "my description",
			Settings: &apis.MapAny{
				Data: map[string]any{
					"queue.type": "persisted",
				},
			},
		},
	}

	expectedPipeline = &kbapi.LogstashPipeline{
		ID:          "test",
		Pipeline:    "input { stdin {} } output { stdout {} }",
		Description: "my description",
		Settings: map[string]any{
			"queue.type": "persisted",
		},
	}

	pipeline, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedPipeline, pipeline)
}
