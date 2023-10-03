package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLicense(t *testing.T) {

	var (
		o        *elasticsearchcrd.Elasticsearch
		s        *corev1.Secret
		licenses []elasticsearchapicrd.License
	)

	// Normal
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			LicenseSecretRef: &corev1.LocalObjectReference{
				Name: "license",
			},
		},
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "license",
		},
		Data: map[string][]byte{
			"license": []byte("license"),
		},
	}

	licenses, err := buildLicenses(o, s)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/license.yml", &licenses[0], test.CleanApi)

	// When no license is expected
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	licenses, err = buildLicenses(o, nil)
	assert.NoError(t, err)
	assert.Empty(t, licenses)
}
