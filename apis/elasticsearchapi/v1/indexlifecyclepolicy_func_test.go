package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIndexLifecyclePolicyGetExternalName(t *testing.T) {
	var o *IndexLifecyclePolicy

	// When name is set
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexLifecyclePolicySpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexLifecyclePolicySpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
