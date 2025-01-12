package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToYamlOrDie(t *testing.T) {
	test := map[string]any{
		"fu": "bar",
	}

	res := ToYamlOrDie(test)
	assert.Equal(t, "fu: bar\n", res)
}
