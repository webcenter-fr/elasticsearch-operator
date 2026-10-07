package kibana

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestKibanaTLSSpecDefaults(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	spec := kibanaTLSSpec(o)

	assert.Equal(t, "test-tls-kb", spec.SecretName)
	assert.Equal(t, "test", spec.CommonName)
	assert.Equal(t, "test-api", spec.CACommonName)
	assert.Len(t, spec.DNSNames, 3)
	assert.Contains(t, spec.DNSNames, "test-kb")
	assert.Contains(t, spec.DNSNames, "test-kb.default")
	assert.Contains(t, spec.DNSNames, "test-kb.default.svc")
	assert.Equal(t, 365, spec.LeafValidityDays)
	assert.Equal(t, 365, spec.CAValidityDays)
	assert.Equal(t, 30, spec.RenewalDays)
	assert.Zero(t, spec.CARenewalDays)
	assert.Equal(t, []string{"test"}, spec.Subject.Organizations)
	assert.Equal(t, []string{"api"}, spec.Subject.OrganizationalUnits)
	assert.Equal(t, []string{"internal"}, spec.Subject.Countries)
	assert.Equal(t, []string{"internal"}, spec.Subject.Localities)
	assert.Equal(t, []string{"internal"}, spec.Subject.Provinces)
	assert.Equal(t, certificate.KeyAlgorithmRSA, spec.KeyAlgorithm)
	assert.Equal(t, 2048, spec.KeySize)
}

func TestKibanaTLSSpecOverrides(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: shared.TlsSpec{
				ValidityDays:  ptr.To[int](180),
				RenewalDays:   ptr.To[int](15),
				CaRenewalDays: ptr.To[int](90),
				KeyComplexity: "ecdsa-p256",
				SelfSignedCertificate: &shared.TlsSelfSignedCertificateSpec{
					AltNames: []string{"extra.example.com"},
					AltIps:   []string{"10.0.0.1"},
				},
			},
		},
	}

	spec := kibanaTLSSpec(o)

	assert.Equal(t, 180, spec.LeafValidityDays)
	assert.Equal(t, 180, spec.CAValidityDays)
	assert.Equal(t, 15, spec.RenewalDays)
	assert.Equal(t, 90, spec.CARenewalDays)
	assert.Contains(t, spec.DNSNames, "extra.example.com")
	assert.Contains(t, spec.IPAddresses, "10.0.0.1")
	assert.Equal(t, certificate.KeyAlgorithmECDSA, spec.KeyAlgorithm)
	assert.Equal(t, certificate.CurveP256, spec.Curve)
}

func TestApplyKeyComplexity(t *testing.T) {
	tests := []struct {
		name          string
		complexity    string
		legacyKeySize *int
		wantAlgo      string
		wantCurve     string
		wantSize      int
	}{
		{"rsa-2048", "rsa-2048", nil, certificate.KeyAlgorithmRSA, "", 2048},
		{"rsa-4096", "rsa-4096", nil, certificate.KeyAlgorithmRSA, "", 4096},
		{"ecdsa-p256", "ecdsa-p256", nil, certificate.KeyAlgorithmECDSA, certificate.CurveP256, 0},
		{"ecdsa-p384", "ecdsa-p384", nil, certificate.KeyAlgorithmECDSA, certificate.CurveP384, 0},
		{"ecdsa-p521", "ecdsa-p521", nil, certificate.KeyAlgorithmECDSA, certificate.CurveP521, 0},
		{"empty-legacy-default", "", nil, certificate.KeyAlgorithmRSA, "", 2048},
		{"empty-legacy-keySize", "", ptr.To[int](4096), certificate.KeyAlgorithmRSA, "", 4096},
		{"invalid-defaults-to-rsa-2048", "unknown-algo", nil, certificate.KeyAlgorithmRSA, "", 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &certificate.TLSSpec{}
			applyKeyComplexity(spec, tt.complexity, tt.legacyKeySize)
			assert.Equal(t, tt.wantAlgo, spec.KeyAlgorithm)
			assert.Equal(t, tt.wantCurve, spec.Curve)
			assert.Equal(t, tt.wantSize, spec.KeySize)
		})
	}
}

func TestCertParse(t *testing.T) {
	// Generate a valid self-signed cert for testing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	assert.NoError(t, err)

	validPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tests := []struct {
		name      string
		input     []byte
		wantError bool
	}{
		{"valid PEM", validPEM, false},
		{"empty input", []byte{}, true},
		{"garbage input", []byte("not a certificate"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crt, err := certParse(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, crt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, crt)
				assert.Equal(t, "test", crt.Subject.CommonName)
			}
		})
	}
}
