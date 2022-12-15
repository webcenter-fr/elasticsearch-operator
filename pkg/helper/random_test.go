package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	a1 := RandomString(5)
	a2 := RandomString(5)
	b1 := RandomString(10)

	assert.NotEqual(t, a1, a2)
	assert.Equal(t, len(a1), len(a2))
	assert.Equal(t, 10, len(b1))
}
