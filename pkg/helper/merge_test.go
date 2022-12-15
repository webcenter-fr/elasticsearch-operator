package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMerge(t *testing.T) {
	var (
		p         *corev1.PodSpec
		p2        *corev1.PodSpec
		expectedP *corev1.PodSpec
		err       error
	)

	// Normal merge
	p = &corev1.PodSpec{
		SchedulerName: "schedule1",
		NodeSelector: map[string]string{
			"app": "test",
			"env": "dev1",
		},
	}

	p2 = &corev1.PodSpec{
		NodeSelector: map[string]string{
			"app": "test2",
			"env": "dev1",
		},
	}

	expectedP = &corev1.PodSpec{
		SchedulerName: "schedule1",
		NodeSelector: map[string]string{
			"app": "test2",
			"env": "dev1",
		},
	}

	err = Merge(p2, p)
	assert.NoError(t, err)
	assert.Equal(t, expectedP, p2)

}
