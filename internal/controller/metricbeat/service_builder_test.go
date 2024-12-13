package metricbeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildServicees(t *testing.T) {
	var (
		err      error
		services []corev1.Service
		o        *beatcrd.Metricbeat
	)

	// With default values
	o = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.MetricbeatSpec{},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	assert.NotEmpty(t, services)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_default.yaml", &services[0], scheme.Scheme)
}
