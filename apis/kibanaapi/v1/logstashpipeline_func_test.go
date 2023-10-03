package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogstashPipelineGetExternalName(t *testing.T) {
	var o *LogstashPipeline

	// When name is set
	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashPipelineSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &LogstashPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashPipelineSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
