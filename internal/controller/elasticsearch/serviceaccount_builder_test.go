package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServiceAccounts(t *testing.T) {
	var (
		err             error
		serviceAccounts []corev1.ServiceAccount
		o               *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, false)
	assert.NoError(t, err)
	assert.Empty(t, serviceAccounts)

	// When on Openshift
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(serviceAccounts))
	test.EqualFromYamlFile[*corev1.ServiceAccount](t, "testdata/serviceaccount_default.yml", &serviceAccounts[0], scheme.Scheme)
}
