package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	assert.Equal(t, "test2", o.GetIndexTemplateName())

	// When name isn't set
	o = &IndexTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexTemplateSpec{},
	}

	assert.Equal(t, "test", o.GetIndexTemplateName())
}