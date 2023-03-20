package elasticsearch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildNetworkPolicy(t *testing.T) {
	var (
		err   error
		o     *elasticsearchcrd.Elasticsearch
		np    *networkingv1.NetworkPolicy
		oList []client.Object
	)

	// When not in pod
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	np, err = BuildNetworkPolicy(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_not_in_pod.yml", np, test.CleanApi)

	// When in pod
	os.Setenv("POD_NAME", "test")
	os.Setenv("POD_NAMESPACE", "test")
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	np, err = BuildNetworkPolicy(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_in_pod.yml", np, test.CleanApi)

	// When in pod and external referers
	os.Setenv("POD_NAME", "test")
	os.Setenv("POD_NAMESPACE", "test")
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	oList = []client.Object{

		&beatcrd.Metricbeat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "metricbeat",
				Namespace: "test",
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "Metricbeat",
			},
		},

		&beatcrd.Filebeat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "filebeat",
				Namespace: "test",
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "Filebeat",
			},
		},
		&kibanacrd.Kibana{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kibana",
				Namespace: "test",
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "Kibana",
			},
		},
		&logstashcrd.Logstash{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "logstash",
				Namespace: "test",
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "Logstash",
			},
		},
	}

	np, err = BuildNetworkPolicy(o, oList)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_referer.yml", np, test.CleanApi)

}
