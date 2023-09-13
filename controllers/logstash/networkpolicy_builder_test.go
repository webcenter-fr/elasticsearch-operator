package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildNetworkPolicies(t *testing.T) {
	var (
		err   error
		o     *logstashcrd.Logstash
		nps   []networkingv1.NetworkPolicy
		oList []client.Object
	)

	// Default
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	nps, err = buildNetworkPolicy(o, nil)

	assert.NoError(t, err)
	assert.Empty(t, nps)

	// When some logstash referer
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name:      "test",
					Namespace: "elastic",
				},
			},
		},
	}
	oList = []client.Object{

		&beatcrd.Filebeat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "filebeat",
				Namespace: "test",
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "Filebeat",
			},
			Spec: beatcrd.FilebeatSpec{
				LogstashRef: beatcrd.FilebeatLogstashRef{
					ManagedLogstashRef: &beatcrd.FilebeatLogstashManagedRef{
						Port: 5003,
					},
				},
			},
		},
	}
	nps, err = buildNetworkPolicy(o, oList)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/networkpolicy_referer.yml", &nps[0], test.CleanApi)
}
