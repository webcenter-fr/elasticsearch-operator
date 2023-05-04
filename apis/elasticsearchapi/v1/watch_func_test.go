package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetWatchName(t *testing.T) {
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

	assert.Equal(t, "test2", o.GetWatchName())

	// When name isn't set
	o = &Watch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: WatchSpec{},
	}

	assert.Equal(t, "test", o.GetWatchName())
}
