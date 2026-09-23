package pki

import (
	"crypto/x509"
	"time"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"
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

// EffectiveCARenewalDays returns the CA renewal window to use, guarding against
// a misconfigured window that is greater than or equal to the CA validity.
// A window >= validity makes a freshly issued CA immediately due for renewal,
// so the rotation saga would rotate it on every reconcile (perpetual renewal
// loop). In that case it falls back to the default 30-day window, capped at
// half the validity for short-lived certificates.
func EffectiveCARenewalDays(caRenewalDays, caValidityDays int) int {
	if caValidityDays > 0 && caRenewalDays >= caValidityDays {
		fallback := 30
		if half := caValidityDays / 2; half < fallback {
			fallback = half
		}
		if fallback < 1 {
			fallback = 1
		}
		return fallback
	}
	return caRenewalDays
}
