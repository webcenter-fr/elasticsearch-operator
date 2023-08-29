package pki

import (
	"net"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	"github.com/sirupsen/logrus"
)

const (
	rootCAApiCN = "elasticsearch-operator.k8s.webcenter.fr"
)

// NewRootCAApi return a CA dedicated for API endpoint (https)
func NewRootCAApi(log *logrus.Entry) (ca *goca.CA, err error) {

	log.Debug("Create new root CA for API Layer")

	rootCAIdentity := goca.Identity{
		Organization:       "Elasticsearch Org",
		OrganizationalUnit: "Certificates Management",
		Country:            "US",
		Locality:           "TORONTO",
		Province:           "ONTARIO",
		Intermediate:       false,
		Valid:              DefaultCertificateValidity,
		KeyBitSize:         KeyBitSize,
	}

	return goca.New(rootCAApiCN, rootCAIdentity)
}

// LoadRootCAApi load existing CA and retun it
func LoadRootCAApi(privateKeyPem []byte, publicKeyPem []byte, certPem []byte, crlPem []byte, log *logrus.Entry) (ca *goca.CA, err error) {

	if privateKeyPem == nil || publicKeyPem == nil || certPem == nil || crlPem == nil {
		return nil, errors.New("You need to provide valide privateKey, publicKey, cert, crl contend")
	}

	log.Debug("Load root CA for API Layer")

	ca = &goca.CA{
		CommonName: rootCAApiCN,
	}

	err = ca.LoadCA(privateKeyPem, publicKeyPem, certPem, crlPem)
	if err != nil {
		return nil, err
	}

	return ca, nil

}

// NewApiTls return certificate dedicated for API endpoint
// Share accross all nodes
func NewApiTls(clusterName string, altNames, altIPs []string, ca *goca.CA, log *logrus.Entry) (certificate *goca.Certificate, err error) {

	if clusterName == "" {
		return nil, errors.New("ClusterName must be provided")
	}
	if ca == nil {
		return nil, errors.New("CA must be provided")
	}

	log.Debugf("Create API certificate for cluster %s", clusterName)
	log.Debugf("With alternative names: %v", altNames)
	log.Debugf("With alternatives IPs: %v", altIPs)

	var ips []net.IP
	dnsNames := []string{clusterName}

	if len(altNames) > 0 {
		dnsNames = append(dnsNames, altNames...)
	}

	if len(altIPs) > 0 {
		ips = make([]net.IP, 0, len(altIPs))
		for _, ipStr := range altIPs {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, errors.Errorf("IP %s is not valid", ipStr)
			}
			ips = append(ips, ip)
		}
	}

	apiIdentity := goca.Identity{
		Organization:       "Elasticsearch Org",
		OrganizationalUnit: "Elasticsearch node",
		Country:            "US",
		Locality:           "TORONTO",
		Province:           "ONTARIO",
		Intermediate:       false,
		DNSNames:           dnsNames,
		IPAddresses:        ips,
		Valid:              DefaultCertificateValidity,
		KeyBitSize:         KeyBitSize,
	}

	return ca.IssueCertificate(clusterName, apiIdentity)
}
