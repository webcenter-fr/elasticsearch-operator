package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIndexTemplateGetStatus(t *testing.T) {
	status := IndexTemplateStatus{
		BasicRemoteObjectStatus: apis.BasicRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestGetIndexTemplateName(t *testing.T) {
	var o *IndexTemplate

	// When name is set
	o = &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexTemplateSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexTemplateSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}

func TestIndexIsRawTemplate(t *testing.T) {
	var o *IndexTemplate

	// When rawTemplate is set
	o = &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexTemplateSpec{
			RawTemplate: "test",
		},
	}

	assert.True(t, o.IsRawTemplate())

	// When raw template is not set
	o = &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexTemplateSpec{},
	}

	assert.False(t, o.IsRawTemplate())
}
