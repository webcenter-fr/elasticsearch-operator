package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildMetricbeat(t *testing.T) {

	var (
		err error
		mb  *beatcrd.Metricbeat
		o   *kibanacrd.Kibana
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	assert.Nil(t, mb)

	// When metricbeat is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
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
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
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
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_default.yaml", mb, test.CleanApi)

	// When metricbeat is enabled with all set
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
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
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_all_set.yaml", mb, test.CleanApi)

	// When metricbeat is enabled and tls is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
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
			Tls: kibanacrd.KibanaTlsSpec{
				Enabled: pointer.Bool(false),
			},
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_tls_disabled.yaml", mb, test.CleanApi)
}
