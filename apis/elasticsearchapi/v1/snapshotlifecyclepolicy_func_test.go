package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSnapshotLifecyclePolicyGetStatus(t *testing.T) {
	status := SnapshotLifecyclePolicyStatus{
		BasicRemoteObjectStatus: apis.BasicRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestSnapshotLifecyclePolicyGetExternalName(t *testing.T) {
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

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &SnapshotLifecyclePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: SnapshotLifecyclePolicySpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
