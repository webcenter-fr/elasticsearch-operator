package pki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiPKI(t *testing.T) {

	// Create CA
	ca, err := NewRootCAApi(testLogEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotEmpty(t, ca.GetCRL())
	assert.NotEmpty(t, ca.GetPublicKey())

	// Create certificate
	crt, err := NewApiTls("test", []string{"elasticsearch.test.local"}, []string{"10.0.0.1"}, ca, testLogEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, crt.GetCertificate())
	assert.NotEmpty(t, crt.PrivateKey)
}
