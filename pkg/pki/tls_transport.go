package pki

import (
	"emperror.dev/errors"
	"github.com/disaster37/goca"
	"github.com/sirupsen/logrus"
)

const rootCATransportCN = "elasticsearch-operator.k8s.webcenter.fr"

// NewRootCATransport return a CA dedicated for transport communication between
// Elasticsearch nodes
func NewRootCATransport(log *logrus.Entry) (ca *goca.CA, err error) {

	log.Debug("Create new root CA for transport Layer")

	rootCAIdentity := goca.Identity{
		Organization:       "Elasticsearch Org",
		OrganizationalUnit: "Certificates Management",
		Country:            "US",
		Locality:           "TORONTO",
		Province:           "ONTARIO",
		Intermediate:       false,
		Valid:              DefaultCertificateValidity, // CA is valid for 1 years
		KeyBitSize:         KeyBitSize,
	}

	return goca.New(rootCATransportCN, rootCAIdentity)
}

// LoadRootCATransport load existing CA and retun it
func LoadRootCATransport(privateKeyPem []byte, publicKeyPem []byte, certPem []byte, crlPem []byte, log *logrus.Entry) (ca *goca.CA, err error) {

	if privateKeyPem == nil || publicKeyPem == nil || certPem == nil || crlPem == nil {
		return nil, errors.New("You need to provide valide privateKey, publicKey, cert, crl contend")
	}

	log.Debug("Load root CA for transport layer")

	ca = &goca.CA{
		CommonName: rootCATransportCN,
	}

	err = ca.LoadCA(privateKeyPem, publicKeyPem, certPem, crlPem)
	if err != nil {
		return nil, err
	}

	return ca, nil

}

// NewNodeTLS return certificate dedicated for node
// Each node must to have his own certificate
func NewNodeTLS(nodeName string, ca *goca.CA, log *logrus.Entry) (certificate *goca.Certificate, err error) {

	if nodeName == "" {
		return nil, errors.New("NodeName must be provided")
	}

	if ca == nil {
		return nil, errors.New("CA must be provided")
	}

	log.Debugf("Create new certificate for node %s", nodeName)

	nodeIdentity := goca.Identity{
		Organization:       "Elasticsearch Org",
		OrganizationalUnit: "Elasticsearch node",
		Country:            "US",
		Locality:           "TORONTO",
		Province:           "ONTARIO",
		Intermediate:       false,
		DNSNames:           []string{nodeName},
		Valid:              DefaultCertificateValidity,
		KeyBitSize:         KeyBitSize,
	}

	return ca.IssueCertificate(nodeName, nodeIdentity)
}
