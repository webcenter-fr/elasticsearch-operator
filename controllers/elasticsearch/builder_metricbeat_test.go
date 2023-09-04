package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildMetricbeat(t *testing.T) {

	var (
		err error
		mb  *beatcrd.Metricbeat
		o   *elasticsearchcrd.Elasticsearch
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	assert.Nil(t, mb)

	// When metricbeat is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
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
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "test",
					Replicas: 1,
				},
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_default.yaml", mb, test.CleanApi)

	// When metricbeat is enabled with all set
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
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
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "test",
					Replicas: 1,
				},
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_all_set.yaml", mb, test.CleanApi)

	// When metricbeat is enabled and tls is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
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
			Tls: elasticsearchcrd.ElasticsearchTlsSpec{
				Enabled: ptr.To[bool](false),
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name:     "test",
					Replicas: 1,
				},
			},
		},
	}

	mb, err = BuildMetricbeat(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/metricbeat_tls_disabled.yaml", mb, test.CleanApi)
}
