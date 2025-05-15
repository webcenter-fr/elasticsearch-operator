package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWatchGetStatus(t *testing.T) {
	status := WatchStatus{
		DefaultRemoteObjectStatus: remote.DefaultRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestWatchGetExternalName(t *testing.T) {
	var o *Watch

	// When name is set
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: WatchSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: WatchSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
