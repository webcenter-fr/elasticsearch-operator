package metricbeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServiceAccounts(t *testing.T) {
	var (
		err             error
		serviceAccounts []corev1.ServiceAccount
		o               *beatcrd.Metricbeat
	)

	// With default values
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, false)
	assert.NoError(t, err)
	assert.Empty(t, serviceAccounts)

	// When on Openshift
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(serviceAccounts))
	test.EqualFromYamlFile[*corev1.ServiceAccount](t, "testdata/serviceaccount_default.yml", &serviceAccounts[0], scheme.Scheme)
}
