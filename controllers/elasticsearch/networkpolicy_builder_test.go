package elasticsearch

import (
	"os"
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildNetworkPolicy(t *testing.T) {
	var (
		err   error
		o     *elasticsearchcrd.Elasticsearch
		nps   []networkingv1.NetworkPolicy
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

	nps, err = buildNetworkPolicies(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_not_in_pod.yml", &nps[0], scheme.Scheme)

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

	nps, err = buildNetworkPolicies(o, nil)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_in_pod.yml", &nps[0], scheme.Scheme)

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

	nps, err = buildNetworkPolicies(o, oList)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_referer.yml", &nps[0], scheme.Scheme)

}
