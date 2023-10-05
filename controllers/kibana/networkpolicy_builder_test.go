package kibana

import (
	"os"
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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

	np, err = buildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(np))
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_not_in_pod.yml", &np[0], scheme.Scheme)

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

	np, err = buildNetworkPolicies(o)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(np))
	test.EqualFromYamlFile[*networkingv1.NetworkPolicy](t, "testdata/networkpolicy_in_pod.yml", &np[0], scheme.Scheme)
}
