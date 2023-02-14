package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZipUnzip(t *testing.T) {
	var (
		err error
		o   map[string]any
		o2  map[string]any
		res string
	)

	// Normale use case
	o = map[string]any{
		"fu": "bar",
	}

	res, err = ZipAndBase64Encode(o)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)

	o2 = map[string]any{}
	err = UnZipBase64Decode(res, &o2)
	assert.NoError(t, err)

	assert.Equal(t, o, o2)

	// When unzip empty string
	o2 = map[string]any{}
	err = UnZipBase64Decode("", &o2)
	assert.NoError(t, err)

}
