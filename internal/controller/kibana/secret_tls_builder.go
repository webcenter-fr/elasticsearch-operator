package kibana

import (
	"fmt"
	"net"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certKeySize  = 2048
	certValidity = 365
)

// BuildPkiSecret generate the secret that store transport PKI
func buildPkiSecret(o *kibanacrd.Kibana) (sPki *corev1.Secret, rootCA *goca.CA, err error) {
	if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
		return nil, nil, nil
	}

	var (
		keySize      int
		validityDays int
	)

	if o.Spec.Tls.ValidityDays != nil {
		validityDays = *o.Spec.Tls.ValidityDays
	} else {
		validityDays = certValidity
	}
	if o.Spec.Tls.KeySize != nil {
		keySize = *o.Spec.Tls.KeySize
	} else {
		keySize = certKeySize
	}

	// Generate new PKI
	rootCAIdentity := goca.Identity{
		Organization:       o.Name,
		OrganizationalUnit: "api",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		Valid:              validityDays,
		KeyBitSize:         keySize,
	}

	rootCA, err = goca.New(fmt.Sprintf("%s-api", o.Name), rootCAIdentity)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error when create PKI")
	}

	sPki = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForPki(o),
			Namespace:   o.Namespace,
			Labels:      getLabels(o),
			Annotations: getAnnotations(o),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": []byte(rootCA.GetCertificate()),
			"ca.key": []byte(rootCA.GetPrivateKey()),
			"ca.pub": []byte(rootCA.GetPublicKey()),
			"ca.crl": []byte(rootCA.GetCRL()),
		},
	}

	return sPki, rootCA, nil
}

// BuildTlsSecret generate the secret that store the http certificates
func buildTlsSecret(o *kibanacrd.Kibana, rootCA *goca.CA) (s *corev1.Secret, err error) {
	if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
		return nil, nil
	}

	crt, err := generateCertificate(o, rootCA)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate certificate")
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForTls(o),
			Namespace:   o.Namespace,
			Labels:      getLabels(o),
			Annotations: getAnnotations(o),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt":  []byte(rootCA.GetCertificate()),
			"tls.crt": []byte(crt.Certificate),
			"tls.key": []byte(crt.RsaPrivateKey),
		},
	}

	return s, nil
}

func generateCertificate(o *kibanacrd.Kibana, rootCA *goca.CA) (nodeCrt *goca.Certificate, err error) {
	var ips []net.IP
	dnsNames := []string{
		GetServiceName(o),
		fmt.Sprintf("%s.%s", GetServiceName(o), o.Namespace),
		fmt.Sprintf("%s.%s.svc", GetServiceName(o), o.Namespace),
	}

	var (
		keySize      int
		validityDays int
	)

	if o.Spec.Tls.ValidityDays != nil {
		validityDays = *o.Spec.Tls.ValidityDays
	} else {
		validityDays = certValidity
	}
	if o.Spec.Tls.KeySize != nil {
		keySize = *o.Spec.Tls.KeySize
	} else {
		keySize = certKeySize
	}

	if o.Spec.Tls.SelfSignedCertificate != nil && len(o.Spec.Tls.SelfSignedCertificate.AltNames) > 0 {
		dnsNames = append(dnsNames, o.Spec.Tls.SelfSignedCertificate.AltNames...)
	}

	if o.Spec.Tls.SelfSignedCertificate != nil && len(o.Spec.Tls.SelfSignedCertificate.AltIps) > 0 {
		ips = make([]net.IP, 0, len(o.Spec.Tls.SelfSignedCertificate.AltIps))
		for _, ipStr := range o.Spec.Tls.SelfSignedCertificate.AltIps {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, errors.Errorf("IP %s is not valid", ipStr)
			}
			ips = append(ips, ip)
		}
	}

	apiIdentity := goca.Identity{
		Organization:       o.Name,
		OrganizationalUnit: "api",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		DNSNames:           dnsNames,
		IPAddresses:        ips,
		Valid:              validityDays,
		KeyBitSize:         keySize,
	}

	return rootCA.IssueCertificate(o.Name, apiIdentity)
}

// updateSecret return true if update existing secret
// It return false if new secret
func updateSecret(o *kibanacrd.Kibana, old, new *corev1.Secret, scheme *runtime.Scheme) (s *corev1.Secret, updated bool, err error) {
	if old != nil {
		old.Labels = new.Labels
		old.Annotations = new.Annotations
		old.Data = new.Data
		updated = true
		s = old
	} else {
		s = new
		updated = false
	}

	// Set ownerReferences on expected object before to diff them
	if err = ctrl.SetControllerReference(o, s, scheme); err != nil {
		return nil, updated, errors.Wrapf(err, "Error when set owner reference on object '%s'", s.GetName())
	}

	return s, updated, nil
}
