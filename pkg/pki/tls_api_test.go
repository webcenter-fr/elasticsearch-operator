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

	// Load CA from pem
	ca2, err := LoadRootCAApi([]byte(ca.GetPrivateKey()), []byte(ca.GetPublicKey()), []byte(ca.GetCertificate()), []byte(ca.GetCRL()), testLogEntry)
	assert.NoError(t, err)
	assert.Equal(t, ca, ca2)

	// When error
	_, err = LoadRootCAApi(nil, nil, nil, nil, nil)
	assert.Error(t, err)

	_, err = NewApiTls("", []string{"elasticsearch.test.local"}, []string{"10.0.0.1"}, ca, testLogEntry)
	assert.Error(t, err)

	_, err = NewApiTls("test", nil, nil, nil, testLogEntry)
	assert.Error(t, err)

}
