package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetComponentTemplateName(t *testing.T) {
	var o *ComponentTemplate

	// When name is set
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ComponentTemplateSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetComponentTemplateName())

	// When name isn't set
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ComponentTemplateSpec{},
	}

	assert.Equal(t, "test", o.GetComponentTemplateName())
}
