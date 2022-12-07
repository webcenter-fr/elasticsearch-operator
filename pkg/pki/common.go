package pki

import (
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DefaultCertificateValidity = 397
	//DefaultRenewCertificate    = -time.Hour * 24 * 7 // 7 days before expired
	KeyBitSize = 2048
)

var testLogEntry = logrus.NewEntry(logrus.New())

// NeedRenewCertificate permit to check if certificate must be renewed before it expire
func NeedRenewCertificate(crt *x509.Certificate, durationBeforeExpire time.Duration, log *logrus.Entry) (status bool, err error) {
	if crt == nil {
		return false, errors.New("Cert must be provided")
	}

	if crt.NotAfter.Before(time.Now().Add(durationBeforeExpire)) {
		log.Debugf("Certificate %s must be renewed, it expire at %s", crt.Subject.CommonName, crt.NotAfter)
		return true, nil
	}

	log.Debugf("Certificate %s not to be renewed, it expire at %s", crt.Subject.CommonName, crt.NotAfter)

	return false, nil
}
