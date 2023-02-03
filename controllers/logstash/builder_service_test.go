package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildServicees(t *testing.T) {

	var (
		err      error
		services []corev1.Service
		o        *logstashcrd.Logstash
	)

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	services, err = BuildServices(o)
	assert.NoError(t, err)
	assert.Empty(t, services)

	// When service is specified
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Services: []logstashcrd.Service{
				{
					Name: "my-service",
					Labels: map[string]string{
						"label1": "value1",
					},
					Annotations: map[string]string{
						"anno1": "value1",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:     "my-port",
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
						Selector: map[string]string{
							"dd": "toto",
						},
					},
				},
			},
		},
	}

	services, err = BuildServices(o)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(services))
	test.EqualFromYamlFile(t, "testdata/service_default.yaml", &services[0], test.CleanApi)

}
