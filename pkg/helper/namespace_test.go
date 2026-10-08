package helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOperatorNamespace(t *testing.T) {
	// Ensure the test is deterministic even when POD_NAMESPACE is already set
	// in the environment, and restore the initial state at the end.
	prevValue, wasSet := os.LookupEnv(operatorNamespaceEnvVar)
	_ = os.Unsetenv(operatorNamespaceEnvVar)
	if wasSet {
		t.Cleanup(func() {
			_ = os.Setenv(operatorNamespaceEnvVar, prevValue)
		})
	}

	// When env not exist
	_, err := GetOperatorNamespace()
	assert.Error(t, err)

	// When env exist
	t.Setenv(operatorNamespaceEnvVar, "test")
	ns, err := GetOperatorNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "test", ns)
}
