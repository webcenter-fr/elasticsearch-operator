package elasticsearch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNetworkPolicy(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		np  *networkingv1.NetworkPolicy
	)

	// When not in pod
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	np, err = BuildNetworkPolicy(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_not_in_pod.yml", np, test.CleanApi)

	// When in pod
	os.Setenv("POD_NAME", "test")
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	np, err = BuildNetworkPolicy(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_in_pod.yml", np, test.CleanApi)

}
