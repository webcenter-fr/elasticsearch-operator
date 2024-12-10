package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsManaged(t *testing.T) {
	var o ElasticsearchRef

	// When managed
	o = ElasticsearchRef{
		ManagedElasticsearchRef: &ElasticsearchManagedRef{
			Name: "test",
		},
	}
	assert.True(t, o.IsManaged())

	// When not managed
	o = ElasticsearchRef{
		ManagedElasticsearchRef: &ElasticsearchManagedRef{},
	}
	assert.False(t, o.IsManaged())

	o = ElasticsearchRef{}
	assert.False(t, o.IsManaged())
}

func TestIsExternal(t *testing.T) {
	var o ElasticsearchRef

	// When external
	o = ElasticsearchRef{
		ExternalElasticsearchRef: &ElasticsearchExternalRef{
			Addresses: []string{
				"test",
			},
		},
	}
	assert.True(t, o.IsExternal())

	// When not managed
	o = ElasticsearchRef{
		ExternalElasticsearchRef: &ElasticsearchExternalRef{},
	}
	assert.False(t, o.IsExternal())

	o = ElasticsearchRef{}
	assert.False(t, o.IsExternal())
}
