package kibana

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPodMonitor(t *testing.T) {
	var (
		err error
		o   *kibanacrd.Kibana
		pm  *monitoringv1.PodMonitor
	)

	// With default values
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	assert.Nil(t, pm)

	// When prometheus monitoring is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
				Prometheus: &kibanacrd.KibanaPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	assert.Nil(t, pm)

	// When prometheus monitoring is enabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: kibanacrd.KibanaMonitoringSpec{
				Prometheus: &kibanacrd.KibanaPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	pm, err = BuildPodMonitor(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/podmonitor.yml", pm, test.CleanApi)

}
