package kibanaapi

import (
	"testing"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLogstashPipeline(t *testing.T) {

	var (
		o                *kibanaapicrd.LogstashPipeline
		pipeline         *kbapi.LogstashPipeline
		expectedPipeline *kbapi.LogstashPipeline
		err              error
	)

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

	pipeline, err = BuildLogstashPipeline(o)
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
			Settings: `
{
	"queue.type": "persisted"
}
			`,
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

	pipeline, err = BuildLogstashPipeline(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedPipeline, pipeline)
}
