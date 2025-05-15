package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestComponentTemplateGetStatus(t *testing.T) {
	status := ComponentTemplateStatus{
		DefaultRemoteObjectStatus: remote.DefaultRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestComponentTemplateGetExternalName(t *testing.T) {
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

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ComponentTemplateSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}

func TestIsRawTemplate(t *testing.T) {
	var o *ComponentTemplate

	// When raw template is set
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ComponentTemplateSpec{
			RawTemplate: ptr.To("test"),
		},
	}

	assert.True(t, o.IsRawTemplate())

	// When raw template is not set
	o = &ComponentTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ComponentTemplateSpec{},
	}

	assert.False(t, o.IsRawTemplate())
}
