package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffDiffLabels(t *testing.T) {
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

	assert.Empty(t, DiffLabels(expectedM, m))

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

	assert.NotEmpty(t, DiffLabels(expectedM, m))

}

func TestDiffAnnotations(t *testing.T) {
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

	assert.Empty(t, DiffAnnotations(expectedM, m))

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

	assert.NotEmpty(t, DiffAnnotations(expectedM, m))

	// When differ but exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"kubectl.kubernetes.io/last-applied-configuration": "dfdf",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"kubectl.kubernetes.io/last-applied-configuration": "aaa",
	}

	assert.Empty(t, DiffAnnotations(expectedM, m))

}
