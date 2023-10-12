package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildPodMonitor(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		pms []monitoringv1.PodMonitor
	)

	sch := scheme.Scheme
	if err := monitoringv1.AddToScheme(sch); err != nil {
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
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	assert.Empty(t, pms)

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
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	assert.Empty(t, pms)

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
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*monitoringv1.PodMonitor](t, "testdata/podmonitor.yml", &pms[0], sch)

}
