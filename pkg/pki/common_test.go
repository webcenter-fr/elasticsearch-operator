package pki

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNeedRenewCertificate(t *testing.T) {

	var (
		cert   *x509.Certificate
		d      time.Duration
		status bool
		err    error
	)

	// When certificate not yet expire
	cert = &x509.Certificate{
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 360),
	}

	d = time.Hour * 24 * 7

	status, err = NeedRenewCertificate(cert, d, testLogEntry)
	assert.NoError(t, err)
	assert.False(t, status)

	// When certificate expire
	cert = &x509.Certificate{
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 360),
	}

	d = time.Hour * 24 * 400

	status, err = NeedRenewCertificate(cert, d, testLogEntry)
	assert.NoError(t, err)
	assert.True(t, status)

	// When certificate is nil
	_, err = NeedRenewCertificate(nil, d, testLogEntry)
	assert.Error(t, err)

}
