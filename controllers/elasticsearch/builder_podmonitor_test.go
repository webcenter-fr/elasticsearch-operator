package elasticsearch

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPodMonitor(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		pm  *monitoringv1.PodMonitor
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	assert.Nil(t, pm)

	// When prometheus monitoring is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
				Prometheus: &elasticsearchcrd.ElasticsearchPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	assert.Nil(t, pm)

	// When prometheus monitoring is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
				Prometheus: &elasticsearchcrd.ElasticsearchPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/podmonitor.yml", pm, test.CleanApi)

}
