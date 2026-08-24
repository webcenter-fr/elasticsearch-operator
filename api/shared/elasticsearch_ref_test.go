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

func TestValidateExternalElasticsearchAddress(t *testing.T) {
	valid := []string{
		"http://elastic.internal:9200",
		"https://elastic.example.com",
		"http://10.0.0.5:9200",
		"http://192.168.1.10:9200",
		"http://127.0.0.1:9200",
		"http://169.254.169.254@evil.com", // userinfo is not the host
	}
	for _, addr := range valid {
		assert.NoError(t, validateExternalElasticsearchAddress(addr), "expected %q to be valid", addr)
	}

	invalid := []string{
		"",                                // empty
		"elastic.internal:9200",           // no scheme
		"ftp://elastic.internal",          // wrong scheme
		"http://",                         // no host
		"http://169.254.169.254:9200",     // metadata endpoint (any port)
		"https://169.254.169.254",         // metadata endpoint
		"http://[::ffff:169.254.169.254]", // IPv4-mapped IPv6 metadata endpoint
		"http://[::ffff:a9fe:a9fe]:9200",  // hex IPv4-mapped metadata endpoint (any port)
		"http://evil.com@169.254.169.254", // userinfo + metadata host
	}
	for _, addr := range invalid {
		assert.Error(t, validateExternalElasticsearchAddress(addr), "expected %q to be invalid", addr)
	}
}

func TestElasticsearchRefValidateFieldExternalAddresses(t *testing.T) {
	// Managed refs are always valid.
	managed := ElasticsearchRef{ManagedElasticsearchRef: &ElasticsearchManagedRef{Name: "test"}}
	assert.Nil(t, managed.ValidateField())

	// Valid external address.
	external := ElasticsearchRef{ExternalElasticsearchRef: &ElasticsearchExternalRef{Addresses: []string{"https://test.local"}}}
	assert.Nil(t, external.ValidateField())

	// Metadata endpoint must be rejected.
	external = ElasticsearchRef{ExternalElasticsearchRef: &ElasticsearchExternalRef{Addresses: []string{"http://169.254.169.254"}}}
	assert.NotNil(t, external.ValidateField())

	// Scheme-less address must be rejected.
	external = ElasticsearchRef{ExternalElasticsearchRef: &ElasticsearchExternalRef{Addresses: []string{"test.local:9200"}}}
	assert.NotNil(t, external.ValidateField())
}
