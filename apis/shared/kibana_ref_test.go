package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKibanaIsManaged(t *testing.T) {
	var o KibanaRef

	// When managed
	o = KibanaRef{
		ManagedKibanaRef: &KibanaManagedRef{
			Name: "test",
		},
	}
	assert.True(t, o.IsManaged())

	// When not managed
	o = KibanaRef{
		ManagedKibanaRef: &KibanaManagedRef{},
	}
	assert.False(t, o.IsManaged())

	o = KibanaRef{}
	assert.False(t, o.IsManaged())

}

func TestKibanaIsExternal(t *testing.T) {
	var o KibanaRef

	// When external
	o = KibanaRef{
		ExternalKibanaRef: &KibanaExternalRef{
			Address: "test",
		},
	}
	assert.True(t, o.IsExternal())

	// When not managed
	o = KibanaRef{
		ExternalKibanaRef: &KibanaExternalRef{},
	}
	assert.False(t, o.IsExternal())

	o = KibanaRef{}
	assert.False(t, o.IsExternal())

}
