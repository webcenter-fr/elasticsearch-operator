package kibana

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				ValidityDays: ptr.To[int](180),
				RenewalDays:  ptr.To[int](15),
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

func TestKibanaCARenewalCustomizer(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	// Helper to generate a CA cert valid for the given duration from now
	generateCACert := func(validFor time.Duration) []byte {
		key, _ := rsa.GenerateKey(rand.Reader, 2048)
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "test-api"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(validFor),
			IsCA:         true,
			BasicConstraintsValid: true,
		}
		certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	}

	t.Run("nil caRenewalDays - unchanged", func(t *testing.T) {
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{},
		}
		c := fake.NewClientBuilder().Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		_, err := customizer.CustomizeCertificate(o, certificate.TLSSpec{})
		assert.NoError(t, err)
		assert.Empty(t, o.Annotations[AnnotationForceRenewTLS])
	})

	t.Run("CA missing - unchanged", func(t *testing.T) {
		caRenewalDays := 30
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{
				Tls: shared.TlsSpec{CaRenewalDays: &caRenewalDays},
			},
		}
		c := fake.NewClientBuilder().Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		_, err := customizer.CustomizeCertificate(o, certificate.TLSSpec{})
		assert.NoError(t, err)
		assert.Empty(t, o.Annotations[AnnotationForceRenewTLS])
	})

	t.Run("CA outside renewal window - unchanged", func(t *testing.T) {
		caRenewalDays := 30
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{
				Tls: shared.TlsSpec{CaRenewalDays: &caRenewalDays},
			},
		}
		caPEM := generateCACert(365 * 24 * time.Hour) // valid for a year
		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls-kb-ca",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"ca.crt": caPEM,
				"ca.key": []byte("fake-key"),
			},
		}
		c := fake.NewClientBuilder().WithObjects(caSecret).Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		_, err := customizer.CustomizeCertificate(o, certificate.TLSSpec{})
		assert.NoError(t, err)
		assert.Empty(t, o.Annotations[AnnotationForceRenewTLS])
	})

	t.Run("CA within renewal window - sets annotation", func(t *testing.T) {
		caRenewalDays := 365 // huge window to guarantee the cert is "within"
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{
				Tls: shared.TlsSpec{CaRenewalDays: &caRenewalDays},
			},
		}
		caPEM := generateCACert(1 * time.Hour) // expires in 1 hour
		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls-kb-ca",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"ca.crt": caPEM,
				"ca.key": []byte("fake-key"),
			},
		}
		c := fake.NewClientBuilder().WithObjects(caSecret).Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		_, err := customizer.CustomizeCertificate(o, certificate.TLSSpec{})
		assert.NoError(t, err)
		assert.Equal(t, "true", o.Annotations[AnnotationForceRenewTLS])
	})

	// Verify the customizer does not mutate the TLSSpec
	t.Run("does not mutate base spec", func(t *testing.T) {
		caRenewalDays := 30
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{
				Tls: shared.TlsSpec{CaRenewalDays: &caRenewalDays},
			},
		}
		c := fake.NewClientBuilder().Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		base := certificate.TLSSpec{SecretName: "original"}
		result, err := customizer.CustomizeCertificate(o, base)
		assert.NoError(t, err)
		assert.Equal(t, "original", result.SecretName)
	})

	// Verify the customizer uses the correct secret name via GetSecretNameForPki
	t.Run("uses correct CA secret name", func(t *testing.T) {
		caRenewalDays := 365
		o := &kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: kibanacrd.KibanaSpec{
				Tls: shared.TlsSpec{CaRenewalDays: &caRenewalDays},
			},
		}
		caPEM := generateCACert(1 * time.Hour)
		caSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tls-kb-ca",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"ca.crt": caPEM,
				"ca.key": []byte("fake-key"),
			},
		}
		c := fake.NewClientBuilder().WithObjects(caSecret).Build()
		customizer := kibanaCARenewalCustomizer(c, log)
		_, err := customizer.CustomizeCertificate(o, certificate.TLSSpec{})
		assert.NoError(t, err)
		assert.Equal(t, "true", o.Annotations[AnnotationForceRenewTLS])

		// Verify the customizer fetched the correct secret
		got := &corev1.Secret{}
		err = c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "test-tls-kb-ca"}, got)
		assert.NoError(t, err)
	})
}
