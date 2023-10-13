package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	var (
		hash string
		err  error
	)

	hash, err = HashPassword("mypassword")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, CheckPasswordHash("mypassword", hash))
	assert.False(t, CheckPasswordHash("fake", hash))
}
