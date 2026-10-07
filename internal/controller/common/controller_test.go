package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnvtest(t *testing.T) {
	t.Run("explicit envtest var", func(t *testing.T) {
		t.Setenv(envtestEnvVar, "true")
		t.Setenv("TEST", "")
		assert.True(t, IsEnvtest())
	})

	t.Run("legacy TEST var", func(t *testing.T) {
		t.Setenv(envtestEnvVar, "")
		t.Setenv("TEST", "true")
		assert.True(t, IsEnvtest())
	})

	t.Run("neither set", func(t *testing.T) {
		t.Setenv(envtestEnvVar, "")
		t.Setenv("TEST", "")
		assert.False(t, IsEnvtest())
	})

	t.Run("non-true values", func(t *testing.T) {
		t.Setenv(envtestEnvVar, "1")
		t.Setenv("TEST", "yes")
		assert.False(t, IsEnvtest())
	})
}
