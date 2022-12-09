package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToSlice(t *testing.T) {

	assert.Equal(t, []string{"test"}, StringToSlice("test", ","))
	assert.Equal(t, []string{}, StringToSlice("", ","))
	assert.Equal(t, []string{"test", "test2"}, StringToSlice("test,test2", ","))
	assert.Equal(t, []string{"test", "test2"}, StringToSlice("test, test2", ","))

}
