package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildMetricbeat(t *testing.T) {

	var (
		err error
		mb  *beatcrd.Metricbeat
		o   *beatcrd.Filebeat
	)

	// With default values
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	assert.Nil(t, mb)

	// When metricbeat is disabled
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Monitoring: beatcrd.FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: false,
				},
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	assert.Nil(t, mb)

	// When metricbeat is enabled with default resource
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Monitoring: beatcrd.FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name:      "test",
							Namespace: "monitoring",
						},
					},
				},
			},
			Deployment: beatcrd.FilebeatDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_default.yaml", mb, test.CleanApi)

	// When metricbeat is enabled with all set
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Monitoring: beatcrd.FilebeatMonitoringSpec{
				Metricbeat: &shared.MetricbeatMonitoringSpec{
					Enabled: true,
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
				},
			},
			Deployment: beatcrd.FilebeatDeploymentSpec{
				Replicas: 3,
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_all_set.yaml", mb, test.CleanApi)
}
