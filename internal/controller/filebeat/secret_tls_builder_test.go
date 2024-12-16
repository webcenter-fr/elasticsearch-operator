package filebeat

import (
	"testing"

	"github.com/disaster37/goca"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildPkiSecret(t *testing.T) {
	var (
		o   *beatcrd.Filebeat
		s   *corev1.Secret
		ca  *goca.CA
		err error
	)

	labels := map[string]string{
		"cluster":                           "test",
		"filebeat.k8s.harmonie-mutuelle.fr": "true",
	}
	annotations := map[string]string{
		"filebeat.k8s.harmonie-mutuelle.fr": "true",
	}

	// When default value
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	s, ca, err = buildPkiSecret(o)
	assert.NoError(t, err)
	assert.NotNil(t, ca)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-pki-fb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))

	// When TLS is enabled
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Pki: beatcrd.FilebeatPkiSpec{
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
	assert.Equal(t, s.Name, "test-pki-fb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))

	// When TLS is disabled
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Pki: beatcrd.FilebeatPkiSpec{
				Enabled: ptr.To[bool](false),
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
		o   *beatcrd.Filebeat
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
		"cluster":                           "test",
		"filebeat.k8s.harmonie-mutuelle.fr": "true",
		"filebeat.k8s.harmonie-mutuelle.fr/tls-certificate": "true",
	}
	annotations := map[string]string{
		"filebeat.k8s.harmonie-mutuelle.fr": "true",
	}

	// When tls is disabled
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Pki: beatcrd.FilebeatPkiSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	s, err = buildTlsSecret(o, ca)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// When tls is enabled
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Pki: beatcrd.FilebeatPkiSpec{
				Enabled: ptr.To[bool](true),
				Tls: map[string]shared.TlsSelfSignedCertificateSpec{
					"test": {
						AltNames: []string{"test.domain.local"},
					},
				},
			},
		},
	}

	s, err = buildTlsSecret(o, ca)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-tls-fb")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.NotEmpty(t, s.Data["ca.crt"])
	assert.NotEmpty(t, s.Data["test.crt"])
	assert.NotEmpty(t, s.Data["test.key"])
}
