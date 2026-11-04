package elasticsearch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildElasticsearchSecretsExternalElasticsearch(t *testing.T) {
	expectedLabels := map[string]string{
		"discoverName":                         "test-elasticsearch",
		discovercrd.ElasticsearchAnnotationKey: "true",
		discovercrd.DiscoverAnnotationKey:      "true",
	}
	expectedEnvAnnotations := map[string]string{
		discovercrd.ElasticsearchAnnotationKey:                         "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-elasticsearch",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "env",
	}
	expectedFileAnnotations := map[string]string{
		discovercrd.ElasticsearchAnnotationKey:                         "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-elasticsearch",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "file",
	}
	secretUserElasticsearch := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-user-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("admin-password"),
		},
	}
	secretCustomCa := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ca-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("custom-ca"),
		},
	}

	// When external Elasticsearch without specified target secrets
	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://elasticsearch1.example.com:9200",
						"https://elasticsearch2.example.com:9200",
					},
				},
			},
		},
	}
	secrets, err := buildElasticsearchSecrets(o, nil, nil, secretUserElasticsearch, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "test-elasticsearch-env", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, expectedLabels, secrets[0].Labels)
	assert.Equal(t, expectedEnvAnnotations, secrets[0].Annotations)
	assert.Equal(t, []byte("https://elasticsearch1.example.com:9200,https://elasticsearch2.example.com:9200"), secrets[0].Data["ELASTICSEARCH_HOSTS_TEST_ELASTICSEARCH"])
	assert.Equal(t, []byte("admin"), secrets[0].Data["ELASTICSEARCH_USERNAME_TEST_ELASTICSEARCH"])
	assert.Equal(t, []byte("admin-password"), secrets[0].Data["ELASTICSEARCH_PASSWORD_TEST_ELASTICSEARCH"])
	assert.Equal(t, "test-elasticsearch-file", secrets[1].Name)
	assert.Equal(t, "default", secrets[1].Namespace)
	assert.Equal(t, expectedLabels, secrets[1].Labels)
	assert.Equal(t, expectedFileAnnotations, secrets[1].Annotations)
	assert.Empty(t, secrets[1].Data["ca.crt"])

	// When user specified target secrets
	o = &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "elasticsearch-env-secret",
				},
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "elasticsearch-file-secret",
				},
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://elasticsearch1.example.com:9200",
					},
				},
			},
		},
	}
	secrets, err = buildElasticsearchSecrets(o, nil, nil, secretUserElasticsearch, nil)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "elasticsearch-env-secret", secrets[0].Name)
	assert.Equal(t, "elasticsearch-file-secret", secrets[1].Name)

	// When user specified custom CA certificate
	o = &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://elasticsearch1.example.com:9200",
					},
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-secret",
				},
			},
		},
	}
	secrets, err = buildElasticsearchSecrets(o, nil, nil, secretUserElasticsearch, secretCustomCa)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, secretCustomCa.Data["ca.crt"], secrets[1].Data["ca.crt"])
}

func TestBuildElasticsearchSecretsManagedElasticsearch(t *testing.T) {
	expectedLabels := map[string]string{
		"discoverName":                         "test-elasticsearch",
		discovercrd.ElasticsearchAnnotationKey: "true",
		discovercrd.DiscoverAnnotationKey:      "true",
	}
	expectedEnvAnnotations := map[string]string{
		discovercrd.ElasticsearchAnnotationKey:                         "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-elasticsearch",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "env",
	}
	expectedFileAnnotations := map[string]string{
		discovercrd.ElasticsearchAnnotationKey:                         "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-elasticsearch",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "file",
	}
	elasticsearchCluster := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-elasticsearch-cluster",
			Namespace: "default",
		},
	}
	secretUserElasticsearch := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elasticsearch-user",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("admin-password"),
		},
	}
	secretCaElasticsearch := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elasticsearch-ca",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("cluster-ca"),
		},
	}
	secretCustomCa := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ca-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("custom-ca"),
		},
	}

	// When managed Elasticsearch without specified target secrets
	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "managed-elasticsearch-cluster",
				},
			},
		},
	}
	secrets, err := buildElasticsearchSecrets(o, elasticsearchCluster, nil, secretUserElasticsearch, nil)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "test-elasticsearch-env", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, expectedLabels, secrets[0].Labels)
	assert.Equal(t, expectedEnvAnnotations, secrets[0].Annotations)
	assert.Equal(t, []byte("https://managed-elasticsearch-cluster-es.default.svc:9200"), secrets[0].Data["ELASTICSEARCH_HOSTS_TEST_ELASTICSEARCH"])
	assert.Equal(t, []byte("admin"), secrets[0].Data["ELASTICSEARCH_USERNAME_TEST_ELASTICSEARCH"])
	assert.Equal(t, []byte("admin-password"), secrets[0].Data["ELASTICSEARCH_PASSWORD_TEST_ELASTICSEARCH"])
	assert.Equal(t, "test-elasticsearch-file", secrets[1].Name)
	assert.Equal(t, "default", secrets[1].Namespace)
	assert.Equal(t, expectedLabels, secrets[1].Labels)
	assert.Equal(t, expectedFileAnnotations, secrets[1].Annotations)

	// When internal PKI is enabled on Elasticsearch cluster
	o = &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "managed-elasticsearch-cluster",
				},
			},
		},
	}
	secrets, err = buildElasticsearchSecrets(o, elasticsearchCluster, secretCaElasticsearch, secretUserElasticsearch, nil)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, secretCaElasticsearch.Data["ca.crt"], secrets[1].Data["ca.crt"])

	// When user specified extra custom CA certificate
	o = &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "managed-elasticsearch-cluster",
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-secret",
				},
			},
		},
	}
	secrets, err = buildElasticsearchSecrets(o, elasticsearchCluster, secretCaElasticsearch, secretUserElasticsearch, secretCustomCa)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, string(secretCaElasticsearch.Data["ca.crt"])+"\n"+string(secretCustomCa.Data["ca.crt"]), string(secrets[1].Data["ca.crt"]))

	// When set env suffix
	o = &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-elasticsearch",
			Namespace: "default",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "managed-elasticsearch-cluster",
				},
			},
			Discover: discovercrd.Discover{
				Name: ptr.To("CUSTOM"),
			},
		},
	}
	secrets, err = buildElasticsearchSecrets(o, elasticsearchCluster, nil, secretUserElasticsearch, nil)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, []byte("https://managed-elasticsearch-cluster-es.default.svc:9200"), secrets[0].Data["ELASTICSEARCH_HOSTS_CUSTOM"])
	assert.NotEmpty(t, secrets[0].Data["ELASTICSEARCH_USERNAME_CUSTOM"])
}
