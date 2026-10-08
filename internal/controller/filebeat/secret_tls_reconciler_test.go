package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	sharedcrd "github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestFilebeatTLSSpec(t *testing.T) {
	// With defaults
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	spec := filebeatTLSSpec(o)
	assert.Equal(t, "test-tls-fb", spec.SecretName)
	assert.Equal(t, "test-filebeat", spec.CACommonName)
	assert.Equal(t, "test", spec.CommonName)
	assert.Equal(t, 365, spec.LeafValidityDays)
	assert.Equal(t, 365, spec.CAValidityDays)
	assert.Equal(t, 30, spec.RenewalDays)
	assert.Equal(t, "RSA", string(spec.KeyAlgorithm))
	assert.Equal(t, 2048, spec.KeySize)
	assert.Equal(t, []string{"test"}, spec.Subject.Organizations)
	assert.Equal(t, []string{"filebeat"}, spec.Subject.OrganizationalUnits)
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

	spec = filebeatTLSSpec(o)
	assert.Equal(t, 730, spec.LeafValidityDays)
	assert.Equal(t, 730, spec.CAValidityDays)
	assert.Equal(t, 60, spec.RenewalDays)
	assert.Equal(t, 4096, spec.KeySize)
}

func TestFilebeatNodeSpecProvider_ExpectedNodeNames(t *testing.T) {
	p := &filebeatNodeSpecProvider{}

	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Pki: beatcrd.FilebeatPkiSpec{
				Enabled: ptr.To[bool](true),
				Tls: map[string]sharedcrd.TlsSelfSignedCertificateSpec{
					"zebra": {},
					"alpha": {},
					"mango": {},
				},
			},
		},
	}

	names, err := p.ExpectedNodeNames(o)
	assert.NoError(t, err)
	// Should be sorted
	assert.Equal(t, []string{"alpha", "mango", "zebra"}, names)
}

func TestFilebeatNodeSpecProvider_NodeCertSpec(t *testing.T) {
	p := &filebeatNodeSpecProvider{}

	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Services: []sharedcrd.Service{
				{Name: "syslog"},
			},
			Pki: beatcrd.FilebeatPkiSpec{
				Enabled: ptr.To[bool](true),
				Tls: map[string]sharedcrd.TlsSelfSignedCertificateSpec{
					"input": {
						AltNames: []string{"custom.example.com", "*.custom.example.com"},
						AltIps:   []string{"10.0.0.1", "127.0.0.1"},
					},
				},
			},
		},
	}

	cn, dnsNames, ips, err := p.NodeCertSpec(o, "input")
	assert.NoError(t, err)
	assert.Equal(t, "input", cn)

	// Check service entries
	assert.Contains(t, dnsNames, "test-syslog-fb")
	assert.Contains(t, dnsNames, "test-syslog-fb.default")
	assert.Contains(t, dnsNames, "test-syslog-fb.default.svc")

	// Check global service entries (Filebeat includes them unlike Logstash)
	assert.Contains(t, dnsNames, "test-headless-fb")
	assert.Contains(t, dnsNames, "test-headless-fb.default")
	assert.Contains(t, dnsNames, "test-headless-fb.default.svc")
	assert.Contains(t, dnsNames, "*.test-headless-fb.default.svc")

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
	o.Spec.Pki.Tls["input"] = sharedcrd.TlsSelfSignedCertificateSpec{
		AltIps: []string{"not-an-ip"},
	}
	_, _, _, err = p.NodeCertSpec(o, "input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IP not-an-ip is not valid")
}
