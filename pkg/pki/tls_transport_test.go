package pki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransportPKI(t *testing.T) {

	// Create CA
	ca, err := NewRootCATransport(testLogEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotEmpty(t, ca.GetCRL())
	assert.NotEmpty(t, ca.GetPublicKey())

	// Create certificate
	crt, err := NewNodeTLS("test", ca, testLogEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, crt.GetCertificate())
	assert.NotEmpty(t, crt.PrivateKey)

	// Load CA
	ca2, err := LoadRootCATransport([]byte(ca.GetPrivateKey()), []byte(ca.GetPublicKey()), []byte(ca.GetCertificate()), []byte(ca.GetCRL()), testLogEntry)
	assert.NoError(t, err)
	assert.Equal(t, ca, ca2)

}
