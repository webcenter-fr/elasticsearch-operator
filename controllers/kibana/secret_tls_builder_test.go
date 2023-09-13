package kibana

import (
	"testing"

	"github.com/disaster37/goca"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildPkiSecret(t *testing.T) {
	var (
		o   *kibanacrd.Kibana
		s   *corev1.Secret
		ca  *goca.CA
		err error
	)

	labels := map[string]string{
		"cluster":                 "test",
		"kibana.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"kibana.k8s.webcenter.fr": "true",
	}

	// When default value
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	s, ca, err = buildPkiSecret(o)
	assert.NoError(t, err)
	assert.NotNil(t, ca)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-pki-kb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))

	// When TLS is enabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](true),
			},
		},
	}

	s, ca, err = buildPkiSecret(o)
	assert.NoError(t, err)
	assert.NotNil(t, ca)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-pki-kb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))

	// When TLS is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	s, ca, err = buildPkiSecret(o)
	assert.NoError(t, err)
	assert.Nil(t, ca)
	assert.Nil(t, s)

	// When Tls is enabled but not self managed
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
		},
	}

	s, ca, err = buildPkiSecret(o)
	assert.NoError(t, err)
	assert.Nil(t, ca)
	assert.Nil(t, s)
}

func TestBuildTlsSecret(t *testing.T) {
	var (
		o   *kibanacrd.Kibana
		s   *corev1.Secret
		err error
	)

	ca, err := goca.NewCA("test", nil, nil, goca.Identity{
		Organization:       "test",
		OrganizationalUnit: "test",
		Country:            "test",
		Locality:           "est",
		Province:           "test",
	})
	if err != nil {
		panic(err)
	}

	labels := map[string]string{
		"cluster":                 "test",
		"kibana.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"kibana.k8s.webcenter.fr": "true",
	}

	// When tls is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	s, err = BuildTlsSecret(o, ca)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// When tls is enabled and not self signed
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
		},
	}

	s, err = BuildTlsSecret(o, ca)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// When tls is enabled and self signed
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: ptr.To[bool](true),
			},
		},
	}

	s, err = BuildTlsSecret(o, ca)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-tls-kb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.NotEmpty(t, s.Data["ca.crt"])
	assert.NotEmpty(t, s.Data["tls.crt"])
	assert.NotEmpty(t, s.Data["tls.key"])
}
