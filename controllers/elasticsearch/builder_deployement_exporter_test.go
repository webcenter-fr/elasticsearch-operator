package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeploymentExporter(t *testing.T) {
	var (
		err error
		o   *elasticsearchcrd.Elasticsearch
		dpl *appv1.Deployment
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	dpl, err = BuildDeploymentExporter(o)
	assert.NoError(t, err)
	assert.Nil(t, dpl)

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
	dpl, err = BuildDeploymentExporter(o)
	assert.NoError(t, err)
	assert.Nil(t, dpl)

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
	dpl, err = BuildDeploymentExporter(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_exporter.yml", dpl, test.CleanApi)
}
