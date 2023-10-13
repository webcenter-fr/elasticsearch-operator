package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsOnStatefulSetUpgradeState(t *testing.T) {
	var o *appv1.StatefulSet

	// When sts is nil
	assert.False(t, IsOnStatefulSetUpgradeState(o))

	// When sts is not on upgrade
	o = &appv1.StatefulSet{}
	assert.False(t, IsOnStatefulSetUpgradeState(o))

	// When generation not the same
	o = &appv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Generation: int64(2),
		},
		Status: appv1.StatefulSetStatus{
			ObservedGeneration: int64(1),
		},
	}
	assert.True(t, IsOnStatefulSetUpgradeState(o))

	// When current revision not the same
	o = &appv1.StatefulSet{
		Status: appv1.StatefulSetStatus{
			CurrentRevision: "1",
			UpdateRevision:  "2",
		},
	}
	assert.True(t, IsOnStatefulSetUpgradeState(o))

	// When replicas not the same
	o = &appv1.StatefulSet{
		Status: appv1.StatefulSetStatus{
			Replicas:      int32(3),
			ReadyReplicas: int32(2),
		},
	}
	assert.True(t, IsOnStatefulSetUpgradeState(o))
}
