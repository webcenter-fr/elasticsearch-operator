package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
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
		o   *elasticsearchcrd.Elasticsearch
	)

	sch := scheme.Scheme
	if err := beatcrd.AddToScheme(sch); err != nil {
		panic(err)
	}

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

	// When metricbeat is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: false,
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	assert.Empty(t, mbs)

	// When metricbeat is enabled with default resource
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: true,
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name:      "test",
							Namespace: "monitoring",
						},
					},
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "test",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_default.yaml", &mbs[0], sch)

	// When metricbeat is enabled with all set
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "test",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_all_set.yaml", &mbs[0], sch)

	// When metricbeat is enabled and tls is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Metricbeat: &shared.MonitoringMetricbeatSpec{
					Enabled: true,
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name:      "test",
							Namespace: "monitoring",
						},
					},
				},
			},
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "test",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	mbs, err = buildMetricbeats(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*beatcrd.Metricbeat](t, "testdata/metricbeat_tls_disabled.yaml", &mbs[0], sch)
}
