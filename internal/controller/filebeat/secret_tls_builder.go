package filebeat

import (
	"fmt"
	"net"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certKeySize  = 2048
	certValidity = 365
)

// BuildPkiSecret generate the secret that store PKI
func buildPkiSecret(o *beatcrd.Filebeat) (sPki *corev1.Secret, rootCA *goca.CA, err error) {
	if !o.Spec.Pki.IsEnabled() {
		return nil, nil, nil
	}

	var (
		keySize      int
		validityDays int
	)

	if o.Spec.Pki.ValidityDays != nil {
		validityDays = *o.Spec.Pki.ValidityDays
	} else {
		validityDays = certValidity
	}
	if o.Spec.Pki.KeySize != nil {
		keySize = *o.Spec.Pki.KeySize
	} else {
		keySize = certKeySize
	}

	// Generate new PKI
	rootCAIdentity := goca.Identity{
		Organization:       o.Name,
		OrganizationalUnit: "filebeat",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		Valid:              validityDays,
		KeyBitSize:         keySize,
	}

	rootCA, err = goca.New(fmt.Sprintf("%s-filebeat", o.Name), rootCAIdentity)
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

// BuildTlsSecret generate the secret that store the certificates
func buildTlsSecret(o *beatcrd.Filebeat, rootCA *goca.CA) (s *corev1.Secret, err error) {
	if !o.Spec.Pki.IsEnabled() {
		return nil, nil
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForTls(o),
			Namespace:   o.Namespace,
			Labels:      getLabelsForTlsSecret(o),
			Annotations: getAnnotations(o),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": []byte(rootCA.GetCertificate()),
		},
	}

	for name, tlsSpec := range o.Spec.Pki.Tls {
		crt, err := generateCertificate(o, rootCA, name, &tlsSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when generate certificate '%s'", name)
		}

		s.Data[fmt.Sprintf("%s.crt", name)] = []byte(crt.Certificate)
		s.Data[fmt.Sprintf("%s.key", name)] = []byte(crt.RsaPrivateKey)
	}

	return s, nil
}

func generateCertificate(o *beatcrd.Filebeat, rootCA *goca.CA, name string, tlsSpec *shared.TlsSelfSignedCertificateSpec) (cert *goca.Certificate, err error) {
	var ips []net.IP
	dnsNames := make([]string, 0, (len(o.Spec.Services)*3)+len(o.Spec.Ingresses))

	// inject services names
	for _, service := range o.Spec.Services {
		dnsNames = append(dnsNames,
			GetServiceName(o, service.Name),
			fmt.Sprintf("%s.%s", GetServiceName(o, service.Name), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetServiceName(o, service.Name), o.Namespace),
			GetGlobalServiceName(o),
			fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
			fmt.Sprintf("*.%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
		)
	}

	// inject ingress names
	for _, ingress := range o.Spec.Ingresses {
		for _, endpoint := range ingress.Spec.Rules {
			dnsNames = append(dnsNames, endpoint.Host)
		}
	}

	// inject custom names
	if tlsSpec.AltNames != nil {
		dnsNames = append(dnsNames, tlsSpec.AltNames...)
	}

	// inject custom IPs
	if len(tlsSpec.AltIps) > 0 {
		ips = make([]net.IP, 0, len(tlsSpec.AltIps))
		for _, ipStr := range tlsSpec.AltIps {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, errors.Errorf("IP %s is not valid", ipStr)
			}
			ips = append(ips, ip)
		}
	}

	var (
		keySize      int
		validityDays int
	)

	if o.Spec.Pki.ValidityDays != nil {
		validityDays = *o.Spec.Pki.ValidityDays
	} else {
		validityDays = certValidity
	}
	if o.Spec.Pki.KeySize != nil {
		keySize = *o.Spec.Pki.KeySize
	} else {
		keySize = certKeySize
	}

	apiIdentity := goca.Identity{
		Organization:       name,
		OrganizationalUnit: "filebeat",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		DNSNames:           dnsNames,
		IPAddresses:        ips,
		Valid:              validityDays,
		KeyBitSize:         keySize,
	}

	return rootCA.IssueCertificate(name, apiIdentity)
}

// updateSecret return true if update existing secret
// It return false if new secret
func updateSecret(o *beatcrd.Filebeat, old, new *corev1.Secret, scheme *runtime.Scheme) (s *corev1.Secret, updated bool, err error) {
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

func getLabelsForTlsSecret(o *beatcrd.Filebeat) map[string]string {
	return getLabels(o, map[string]string{
		fmt.Sprintf("%s/tls-certificate", beatcrd.FilebeatAnnotationKey): "true",
	})
}
