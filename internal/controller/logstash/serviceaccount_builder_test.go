package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServiceAccounts(t *testing.T) {
	var (
		err             error
		serviceAccounts []*corev1.ServiceAccount
		o               *logstashcrd.Logstash
	)

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, false)
	assert.NoError(t, err)
	assert.Empty(t, serviceAccounts)

	// When on Openshift
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	serviceAccounts, err = buildServiceAccounts(o, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(serviceAccounts))
	test.EqualFromYamlFile[*corev1.ServiceAccount](t, "testdata/serviceaccount_default.yml", serviceAccounts[0], scheme.Scheme)
}
