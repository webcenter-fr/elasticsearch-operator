package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSnapshotLifecyclePolicyName(t *testing.T) {
	var o *SnapshotLifecyclePolicy

	// When name is set
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: SnapshotLifecyclePolicySpec{
			SnapshotLifecyclePolicyName: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetSnapshotLifecyclePolicyName())

	// When name isn't set
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: SnapshotLifecyclePolicySpec{},
	}

	assert.Equal(t, "test", o.GetSnapshotLifecyclePolicyName())
}
