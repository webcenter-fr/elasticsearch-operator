package kibana

import (
	"fmt"
	"net"

	"github.com/disaster37/goca"
	"github.com/pkg/errors"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	certKeySize  = 2048
	certValidity = 397
)

// BuildPkiSecret generate the secret that store transport PKI
func BuildPkiSecret(o *kibanaapi.Kibana) (sPki *corev1.Secret, rootCA *goca.CA, err error) {

	if !o.IsTlsEnabled() || !o.IsSelfManagedSecretForTls() {
		return nil, nil, nil
	}

	// Generate new PKI
	rootCAIdentity := goca.Identity{
		Organization:       o.Name,
		OrganizationalUnit: "api",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		Valid:              certValidity,
		KeyBitSize:         certKeySize,
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
func BuildTlsSecret(o *kibanaapi.Kibana, rootCA *goca.CA) (s *corev1.Secret, err error) {

	if !o.IsTlsEnabled() || !o.IsSelfManagedSecretForTls() {
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

func generateCertificate(o *kibanaapi.Kibana, rootCA *goca.CA) (nodeCrt *goca.Certificate, err error) {

	var ips []net.IP
	dnsNames := []string{}

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
		Valid:              certValidity,
		KeyBitSize:         certKeySize,
	}

	return rootCA.IssueCertificate(o.Name, apiIdentity)
}

// updateSecret return true if update existing secret
// It return false if new secret
func updateSecret(old, new *corev1.Secret) (s *corev1.Secret, updated bool) {
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

	return s, updated
}
