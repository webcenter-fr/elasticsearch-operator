package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestIndexLifecyclePolicyGetStatus(t *testing.T) {
	status := IndexLifecyclePolicyStatus{
		DefaultRemoteObjectStatus: remote.DefaultRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

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

func TestIndexStateManagemenIsRawPolicy(t *testing.T) {
	var o *IndexLifecyclePolicy

	// When not raw policy
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexLifecyclePolicySpec{},
	}

	assert.False(t, o.IsRawPolicy())

	// When is raw policy
	o = &IndexLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: IndexLifecyclePolicySpec{
			RawPolicy: ptr.To("test"),
		},
	}

	assert.True(t, o.IsRawPolicy())
}
