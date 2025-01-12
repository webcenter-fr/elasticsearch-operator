package logstash

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildMetricbeat(t *testing.T) {
	var (
		err error
		mbs []beatcrd.Metricbeat
		o   *logstashcrd.Logstash
	)

	sch := scheme.Scheme
	if err := beatcrd.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

	// When metricbeat is disabled
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: ptr.To[bool](false),
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

	// When metricbeat is enabled with default resource
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: ptr.To(true),
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name:      "test",
							Namespace: "monitoring",
						},
					},
				},
			},
			Deployment: logstashcrd.LogstashDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_default.yaml", &mbs[0], sch)

	// When metricbeat is enabled with all set
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: ptr.To(true),
					Version: "1.0.0",
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name:      "test",
							Namespace: "monitoring",
						},
					},
					RefreshPeriod: "5s",
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("200Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("400m"),
							corev1.ResourceMemory: resource.MustParse("300Mi"),
						},
					},
					NumberOfReplica: 1,
				},
			},
			Deployment: logstashcrd.LogstashDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 3,
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_all_set.yaml", &mbs[0], sch)
}
