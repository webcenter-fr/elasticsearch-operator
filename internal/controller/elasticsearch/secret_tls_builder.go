package elasticsearch

import (
	"fmt"
	"net"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certKeySize  = 2048
	certValidity = 397
)

// buildTransportPkiSecret generate the secret that store transport PKI
func buildTransportPkiSecret(o *elasticsearchcrd.Elasticsearch) (sPki *corev1.Secret, rootCA *goca.CA, err error) {
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
		OrganizationalUnit: "transport",
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		Valid:              validityDays,
		KeyBitSize:         keySize,
	}

	rootCA, err = goca.New(fmt.Sprintf("%s-transport", o.Name), rootCAIdentity)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error when create transport PKI")
	}

	sPki = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForPkiTransport(o),
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

// buildTransportSecret generate the secret that store the node certificates
// Add annotations to keep sequence version to know if need to rolling restart nodeGroup (statefullset) on statefullset controller
func buildTransportSecret(o *elasticsearchcrd.Elasticsearch, rootCA *goca.CA) (s *corev1.Secret, err error) {
	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForTlsTransport(o),
			Namespace: o.Namespace,
			Labels:    getLabels(o),
			Annotations: getAnnotations(o, map[string]string{
				fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey): helper.RandomString(64),
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": []byte(rootCA.GetCertificate()),
		},
	}

	// Generate nodes certificates
	for _, nodeGroup := range o.Spec.NodeGroups {
		// Get already existing pod to have static IP
		// Don't generate node certificate for pod that not yet IP
		// Delete existing pods
		for _, nodeName := range GetNodeGroupNodeNames(o, nodeGroup.Name) {
			nodeCrt, err := generateNodeCertificate(o, nodeGroup.Name, nodeName, rootCA)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when generate node certificate for %s", nodeName)
			}
			s.Data[fmt.Sprintf("%s.crt", nodeName)] = []byte(nodeCrt.Certificate)
			s.Data[fmt.Sprintf("%s.key", nodeName)] = []byte(nodeCrt.RsaPrivateKey)
		}
	}

	return s, nil
}

// buildApiPkiSecret generate the secret that store API PKI
func buildApiPkiSecret(o *elasticsearchcrd.Elasticsearch) (sPki *corev1.Secret, rootCA *goca.CA, err error) {
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
		return nil, nil, errors.Wrap(err, "Error when create API PKI")
	}

	sPki = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForPkiApi(o),
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

// buildApiSecret generate the secret that store the API certificate
func buildApiSecret(o *elasticsearchcrd.Elasticsearch, rootCA *goca.CA) (s *corev1.Secret, err error) {
	if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
		return nil, nil
	}

	apiCrt, err := generateApiCertificate(o, rootCA)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate API certificate")
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForTlsApi(o),
			Namespace: o.Namespace,
			Labels:    getLabels(o),
			Annotations: getAnnotations(o, map[string]string{
				fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey): helper.RandomString(64),
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt":  []byte(rootCA.GetCertificate()),
			"tls.crt": []byte(apiCrt.Certificate),
			"tls.key": []byte(apiCrt.RsaPrivateKey),
		},
	}

	return s, nil
}

func generateNodeCertificate(o *elasticsearchcrd.Elasticsearch, nodeGroupName, nodeName string, rootCA *goca.CA) (nodeCrt *goca.Certificate, err error) {
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

	// Generate nodes certificates
	apiIdentity := goca.Identity{
		Organization:       o.Name,
		OrganizationalUnit: nodeGroupName,
		Country:            "internal",
		Locality:           "internal",
		Province:           "internal",
		Intermediate:       false,
		DNSNames: []string{
			fmt.Sprintf("%s.%s", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName)),
			fmt.Sprintf("%s.%s.%s", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
			fmt.Sprintf("%s.%s.%s.svc", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
			GetNodeGroupServiceName(o, nodeGroupName),
			fmt.Sprintf("%s.%s", GetNodeGroupServiceName(o, nodeGroupName), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetNodeGroupServiceName(o, nodeGroupName), o.Namespace),
			GetNodeGroupServiceNameHeadless(o, nodeGroupName),
			fmt.Sprintf("%s.%s", GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
			nodeName,
			fmt.Sprintf("%s.%s", nodeName, o.Namespace),
			fmt.Sprintf("%s.%s.svc", nodeName, o.Namespace),
			GetGlobalServiceName(o),
			fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
		},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
		},
		Valid:      validityDays,
		KeyBitSize: keySize,
	}

	return rootCA.IssueCertificate(nodeName, apiIdentity)
}

func generateApiCertificate(o *elasticsearchcrd.Elasticsearch, rootCA *goca.CA) (nodeCrt *goca.Certificate, err error) {
	var ips []net.IP
	dnsNames := []string{
		GetGlobalServiceName(o),
		fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
		fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
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

	for _, nodeGroup := range o.Spec.NodeGroups {
		dnsNames = append(dnsNames,
			GetNodeGroupServiceName(o, nodeGroup.Name),
			fmt.Sprintf("%s.%s", GetNodeGroupServiceName(o, nodeGroup.Name), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetNodeGroupServiceName(o, nodeGroup.Name), o.Namespace),
		)
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
func updateSecret(o *elasticsearchcrd.Elasticsearch, old, new *corev1.Secret, scheme *runtime.Scheme) (s *corev1.Secret, updated bool, err error) {
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

func getLabelsForTlsSecret(o *elasticsearchcrd.Elasticsearch) map[string]string {
	return getLabels(o, map[string]string{
		fmt.Sprintf("%s/tls-certificate", elasticsearchcrd.ElasticsearchAnnotationKey): "true",
	})
}
