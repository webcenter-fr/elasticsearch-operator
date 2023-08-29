package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	// When prometheus monitoring is enabled and set image version and requirements
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: elasticsearchcrd.ElasticsearchMonitoringSpec{
				Prometheus: &elasticsearchcrd.ElasticsearchPrometheusSpec{
					Enabled: true,
					Version: "v1.6.0",
					Resources: &v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("500m"),
							v1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1000m"),
							v1.ResourceMemory: resource.MustParse("1024Mi"),
						},
					},
				},
			},
		},
	}
	dpl, err = BuildDeploymentExporter(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_exporter_resources.yml", dpl, test.CleanApi)
}
