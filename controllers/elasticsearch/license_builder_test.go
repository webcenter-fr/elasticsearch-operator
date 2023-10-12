package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildLicense(t *testing.T) {

	var (
		o        *elasticsearchcrd.Elasticsearch
		s        *corev1.Secret
		licenses []elasticsearchapicrd.License
	)
	sch := scheme.Scheme
	if err := elasticsearchapicrd.AddToScheme(sch); err != nil {
		panic(err)
	}

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
	test.EqualFromYamlFile[*elasticsearchapicrd.License](t, "testdata/license.yml", &licenses[0], sch)

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
