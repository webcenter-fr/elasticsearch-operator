package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
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
		o   *kibanacrd.Kibana
	)

	sch := scheme.Scheme
	if err := beatcrd.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

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

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

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

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_default.yaml", &mbs[0], sch)

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
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_all_set.yaml", &mbs[0], sch)

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
				Enabled: ptr.To[bool](false),
			},
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_tls_disabled.yaml", &mbs[0], sch)
}
