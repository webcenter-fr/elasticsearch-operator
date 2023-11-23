package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestIsSelfManagedSecretForTls(t *testing.T) {
	var o TlsSpec

	// With default settings
	o = TlsSpec{}
	assert.True(t, o.IsSelfManagedSecretForTls())

	// When TLS is enabled but without specify secrets
	o = TlsSpec{
		Enabled: ptr.To[bool](true),
	}
	assert.True(t, o.IsSelfManagedSecretForTls())

	// When TLS is enabled and pecify secrets
	o = TlsSpec{
		Enabled: ptr.To[bool](true),
		CertificateSecretRef: &corev1.LocalObjectReference{
			Name: "my-secret",
		},
	}
	assert.False(t, o.IsSelfManagedSecretForTls())

}

func TestIsTlsEnabled(t *testing.T) {
	var o TlsSpec

	// With default values
	o = TlsSpec{}
	assert.True(t, o.IsTlsEnabled())

	// When enabled
	o = TlsSpec{
		Enabled: ptr.To[bool](true),
	}
	assert.True(t, o.IsTlsEnabled())

	// When disabled
	o = TlsSpec{
		Enabled: ptr.To[bool](false),
	}
	assert.False(t, o.IsTlsEnabled())
}
