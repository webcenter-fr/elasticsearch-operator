package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	sharedcrd "github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestLogstashTLSSpec(t *testing.T) {
	// With defaults
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	spec := logstashTLSSpec(o)
	assert.Equal(t, "test-tls-ls", spec.SecretName)
	assert.Equal(t, "test-logstash", spec.CACommonName)
	assert.Equal(t, "test", spec.CommonName)
	assert.Equal(t, 365, spec.LeafValidityDays)
	assert.Equal(t, 365, spec.CAValidityDays)
	assert.Equal(t, 30, spec.RenewalDays)
	assert.Equal(t, "RSA", string(spec.KeyAlgorithm))
	assert.Equal(t, 2048, spec.KeySize)
	assert.Equal(t, []string{"test"}, spec.Subject.Organizations)
	assert.Equal(t, []string{"logstash"}, spec.Subject.OrganizationalUnits)
	assert.Equal(t, []string{"internal"}, spec.Subject.Countries)
	assert.Equal(t, []string{"internal"}, spec.Subject.Localities)
	assert.Equal(t, []string{"internal"}, spec.Subject.Provinces)

	// With overrides
	validityDays := 730
	renewalDays := 60
	keySize := 4096
	o.Spec.Pki.ValidityDays = &validityDays
	o.Spec.Pki.RenewalDays = &renewalDays
	o.Spec.Pki.KeySize = &keySize

	spec = logstashTLSSpec(o)
	assert.Equal(t, 730, spec.LeafValidityDays)
	assert.Equal(t, 730, spec.CAValidityDays)
	assert.Equal(t, 60, spec.RenewalDays)
	assert.Equal(t, 4096, spec.KeySize)
}

func TestLogstashNodeSpecProvider_ExpectedNodeNames(t *testing.T) {
	p := &logstashNodeSpecProvider{}

	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Pki: logstashcrd.LogstashPkiSpec{
				Enabled: ptr.To[bool](true),
				Tls: map[string]logstashcrd.LogstashTlsSpec{
					"zebra":   {},
					"alpha":   {},
					"charlie": {},
				},
			},
		},
	}

	names, err := p.ExpectedNodeNames(o)
	assert.NoError(t, err)
	// Should be sorted
	assert.Equal(t, []string{"alpha", "charlie", "zebra"}, names)
}

func TestLogstashNodeSpecProvider_NodeCertSpec(t *testing.T) {
	p := &logstashNodeSpecProvider{}

	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Services: []sharedcrd.Service{
				{Name: "beats"},
			},
			Pki: logstashcrd.LogstashPkiSpec{
				Enabled: ptr.To[bool](true),
				Tls: map[string]logstashcrd.LogstashTlsSpec{
					"beats-input": {
						Consumer: "beat",
						TlsSelfSignedCertificateSpec: sharedcrd.TlsSelfSignedCertificateSpec{
							AltNames: []string{"custom.example.com", "*.custom.example.com"},
							AltIps:   []string{"10.0.0.1", "127.0.0.1"},
						},
					},
				},
			},
		},
	}

	cn, dnsNames, ips, err := p.NodeCertSpec(o, "beats-input")
	assert.NoError(t, err)
	assert.Equal(t, "beats-input", cn)

	// Check service entries
	assert.Contains(t, dnsNames, "test-beats-ls")
	assert.Contains(t, dnsNames, "test-beats-ls.default")
	assert.Contains(t, dnsNames, "test-beats-ls.default.svc")

	// Check that Logstash does NOT include global service entries
	// (unlike Filebeat - old Logstash generateCertificate only adds per-service)
	assert.NotContains(t, dnsNames, "test-headless-ls")
	assert.NotContains(t, dnsNames, "test-headless-ls.default")

	// Check altNames
	assert.Contains(t, dnsNames, "custom.example.com")
	assert.Contains(t, dnsNames, "*.custom.example.com")

	// Check IPs
	assert.Equal(t, []string{"10.0.0.1", "127.0.0.1"}, ips)

	// Test unknown node name
	_, _, _, err = p.NodeCertSpec(o, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown TLS entry")

	// Test invalid IP
	o.Spec.Pki.Tls["beats-input"] = logstashcrd.LogstashTlsSpec{
		TlsSelfSignedCertificateSpec: sharedcrd.TlsSelfSignedCertificateSpec{
			AltIps: []string{"not-an-ip"},
		},
	}
	_, _, _, err = p.NodeCertSpec(o, "beats-input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IP not-an-ip is not valid")
}
