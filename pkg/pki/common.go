package pki

import (
	"crypto/x509"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	"github.com/sirupsen/logrus"
)

const (
	DefaultCertificateValidity = 397
	// DefaultRenewCertificate    = -time.Hour * 24 * 7 // 7 days before expired
	KeyBitSize = 2048
	rootCACN   = "elasticsearch-operator.k8s.webcenter.fr"
)

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

// LoadRootCA load existing CA and retun it
func LoadRootCA(privateKeyPem []byte, publicKeyPem []byte, certPem []byte, crlPem []byte, log *logrus.Entry) (ca *goca.CA, err error) {
	if privateKeyPem == nil || publicKeyPem == nil || certPem == nil || crlPem == nil {
		return nil, errors.New("You need to provide valide privateKey, publicKey, cert, crl contend")
	}

	log.Debug("Load root CA for transport layer")

	ca = &goca.CA{
		CommonName: rootCACN,
	}

	err = ca.LoadCA(privateKeyPem, publicKeyPem, certPem, crlPem)
	if err != nil {
		return nil, err
	}

	return ca, nil
}
