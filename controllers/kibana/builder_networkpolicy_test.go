package kibana

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNetworkPolicies(t *testing.T) {
	var (
		err error
		o   *kibanacrd.Kibana
		np  []networkingv1.NetworkPolicy
	)

	// When not in pod
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	np, err = BuildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(np))
	test.EqualFromYamlFile(t, "testdata/networkpolicy_not_in_pod.yml", &np[0], test.CleanApi)

	// When in pod
	os.Setenv("POD_NAME", "test")
	os.Setenv("POD_NAMESPACE", "test")
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	np, err = BuildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(np))
	test.EqualFromYamlFile(t, "testdata/networkpolicy_in_pod.yml", &np[0], test.CleanApi)

	// When Elasticsearch is managed and in separate namespace
	os.Setenv("POD_NAME", "test")
	os.Setenv("POD_NAMESPACE", "test")
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
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
	assert.Equal(t, 2, len(np))
	test.EqualFromYamlFile(t, "testdata/networkpolicy_es.yml", &np[1], test.CleanApi)
}
