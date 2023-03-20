package metricbeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	services, err = BuildServices(o)
	assert.NoError(t, err)
	assert.NotEmpty(t, services)
	test.EqualFromYamlFile(t, "testdata/service_default.yaml", &services[0], test.CleanApi)

}
