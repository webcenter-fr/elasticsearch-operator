package pki

import (
	"crypto/x509"
	"testing"
	"time"

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

func TestEffectiveCARenewalDays(t *testing.T) {
	tests := []struct {
		name           string
		caRenewalDays  int
		caValidityDays int
		want           int
	}{
		{"window equals validity falls back to default", 365, 365, 30},
		{"default window below validity is kept", 30, 365, 30},
		{"custom window below validity is kept", 300, 365, 300},
		{"short-lived cert caps fallback at half validity", 10, 10, 5},
		{"one-day cert clamps fallback to one", 1, 1, 1},
		{"unknown validity keeps configured window", 30, 0, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EffectiveCARenewalDays(tt.caRenewalDays, tt.caValidityDays))
		})
	}
}
