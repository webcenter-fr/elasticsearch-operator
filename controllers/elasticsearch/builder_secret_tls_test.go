package elasticsearch

import (
	"testing"

	"github.com/disaster37/goca"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildTransportPkiSecret(t *testing.T) {
	var (
		o   *elasticsearchcrd.Elasticsearch
		s   *corev1.Secret
		ca  *goca.CA
		err error
	)

	labels := map[string]string{
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	// When only one node groups
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, ca, err = BuildTransportPkiSecret(o)
	assert.NoError(t, err)
	assert.NotNil(t, ca)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-pki-transport-es")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))
}

func TestBuildApiPkiSecret(t *testing.T) {
	var (
		o   *elasticsearchcrd.Elasticsearch
		s   *corev1.Secret
		ca  *goca.CA
		err error
	)

	labels := map[string]string{
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	// When Tls API is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(false),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, ca, err = BuildApiPkiSecret(o)
	assert.NoError(t, err)
	assert.Nil(t, ca)
	assert.Nil(t, s)

	// When Tls API is enabled but not self managed
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, ca, err = BuildApiPkiSecret(o)
	assert.NoError(t, err)
	assert.Nil(t, ca)
	assert.Nil(t, s)

	// When Tls API is enabled and self managed
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(true),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, ca, err = BuildApiPkiSecret(o)
	assert.NoError(t, err)
	assert.NotNil(t, ca)
	assert.NotEmpty(t, ca.GetCertificate())
	assert.NotEmpty(t, ca.GetPrivateKey())
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-pki-api-es")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.Equal(t, ca.GetCertificate(), string(s.Data["ca.crt"]))
	assert.Equal(t, ca.GetPrivateKey(), string(s.Data["ca.key"]))
	assert.Equal(t, ca.GetPublicKey(), string(s.Data["ca.pub"]))
	assert.Equal(t, ca.GetCRL(), string(s.Data["ca.crl"]))
}

func TestBuildTransportSecret(t *testing.T) {
	var (
		o   *elasticsearchcrd.Elasticsearch
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
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	// When only one node groups
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, err = BuildTransportSecret(o, ca)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-tls-transport-es")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.NotEmpty(t, s.Data["ca.crt"])
	assert.NotEmpty(t, s.Data["test-master-es-0.crt"])
	assert.NotEmpty(t, s.Data["test-master-es-0.key"])
	assert.NotEmpty(t, s.Data["test-master-es-1.crt"])
	assert.NotEmpty(t, s.Data["test-master-es-1.key"])
	assert.NotEmpty(t, s.Data["test-master-es-2.crt"])
	assert.NotEmpty(t, s.Data["test-master-es-2.key"])
}

func TestBuildApiSecret(t *testing.T) {
	var (
		o   *elasticsearchcrd.Elasticsearch
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
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}
	annotations := map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	// When API tls is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(false),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, err = BuildApiSecret(o, ca)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// When API tls is enabled and not self signed
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, err = BuildApiSecret(o, ca)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// When API tls is enabled and self signed
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: pointer.Bool(true),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "master",
					Replicas: 3,
				},
			},
		},
	}

	s, err = BuildApiSecret(o, ca)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s.Name, "test-tls-api-es")
	assert.Equal(t, s.Namespace, "default")
	assert.Equal(t, s.Labels, labels)
	assert.Equal(t, s.Annotations, annotations)
	assert.NotEmpty(t, s.Data["ca.crt"])
	assert.NotEmpty(t, s.Data["tls.crt"])
	assert.NotEmpty(t, s.Data["tls.key"])
}
