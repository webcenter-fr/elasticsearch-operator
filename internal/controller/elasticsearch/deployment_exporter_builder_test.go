package elasticsearch

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildDeploymentExporter(t *testing.T) {
	var (
		err  error
		o    *elasticsearchcrd.Elasticsearch
		dpls []*appv1.Deployment
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	dpls, err = buildDeploymentExporters(o)
	assert.NoError(t, err)
	assert.Empty(t, dpls)

	// When prometheus monitoring is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: ptr.To(false),
				},
			},
		},
	}
	dpls, err = buildDeploymentExporters(o)
	assert.NoError(t, err)
	assert.Empty(t, dpls)

	// When prometheus monitoring is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: ptr.To(true),
				},
			},
		},
	}
	dpls, err = buildDeploymentExporters(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_exporter.yml", dpls[0], scheme.Scheme)

	// When prometheus monitoring is enabled and set image version and requirements
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: ptr.To(true),
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
	dpls, err = buildDeploymentExporters(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_exporter_resources.yml", dpls[0], scheme.Scheme)
}
