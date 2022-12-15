package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestDiff(t *testing.T) {
	var (
		p         *corev1.PodSpec
		expectedP *corev1.PodSpec
	)

	// When same

	p = &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "vol1",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol3",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	expectedP = &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "vol3",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol1",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	assert.Equal(t, "", Diff(expectedP, p))

	// When not the same
	p = &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "vol1",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol3",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	expectedP = &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "vol3",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol2",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "vol4",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	assert.NotEmpty(t, Diff(expectedP, p))

}

func TestDiffMapString(t *testing.T) {
	var (
		m         map[string]string
		expectedM map[string]string
	)

	// When the same without exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	assert.Empty(t, DiffMapString(expectedM, m, nil))

	// When differ

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val4",
	}

	assert.NotEmpty(t, DiffMapString(expectedM, m, nil))

	// When differ but exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val4",
	}

	assert.Empty(t, DiffMapString(expectedM, m, []string{"key3"}))

}
