package pki

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/disaster37/goca"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var testLogEntry = logrus.NewEntry(logrus.New())

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

func TestLoadRootCA(t *testing.T) {
	rootCAIdentity := goca.Identity{
		Organization:       "test",
		OrganizationalUnit: "test",
		Country:            "test",
		Locality:           "test",
		Province:           "test",
		Intermediate:       false,
		Valid:              DefaultCertificateValidity,
		KeyBitSize:         KeyBitSize,
	}

	ca, err := goca.New(rootCACN, rootCAIdentity)
	if err != nil {
		t.Fatal(err)
	}

	// Load CA
	ca2, err := LoadRootCA([]byte(ca.GetPrivateKey()), []byte(ca.GetPublicKey()), []byte(ca.GetCertificate()), []byte(ca.GetCRL()), testLogEntry)
	assert.NoError(t, err)
	assert.Equal(t, ca, ca2)

	// When errors
	_, err = LoadRootCA(nil, nil, nil, nil, nil)
	assert.Error(t, err)
}
