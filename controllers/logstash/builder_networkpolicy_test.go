package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNetworkPolicies(t *testing.T) {
	var (
		err error
		o   *logstashcrd.Logstash
		np  []networkingv1.NetworkPolicy
	)

	// When not in pod
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	np, err = BuildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Empty(t, np)

	// When Elasticsearch is managed
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
	np, err = BuildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(np))
	test.EqualFromYamlFile(t, "testdata/networkpolicy_es.yml", &np[0], test.CleanApi)
}
