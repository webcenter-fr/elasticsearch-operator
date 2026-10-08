package helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOperatorNamespace(t *testing.T) {
	// When env not exist
	_, err := GetOperatorNamespace()
	assert.Error(t, err)

	// When env exist
	_ = os.Setenv(operatorNamespaceEnvVar, "test")
	ns, err := GetOperatorNamespace()
	assert.NoError(t, err)
	assert.Equal(t, "test", ns)
}
