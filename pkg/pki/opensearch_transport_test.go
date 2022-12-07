package pki

import (
	"testing"
	"time"

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

	status, err := NeedRenewCertificate(crt.GoCACertificate(), -time.Hour*24*7, testLogEntry)
	assert.NoError(t, err)
	assert.False(t, status)
}
