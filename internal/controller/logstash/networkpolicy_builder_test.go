package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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

	nps, err = buildNetworkPolicies(o, nil)

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
	nps, err = buildNetworkPolicies(o, oList)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_referer.yml", &nps[0], scheme.Scheme)
}
