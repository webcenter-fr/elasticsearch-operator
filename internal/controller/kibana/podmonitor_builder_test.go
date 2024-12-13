package kibana

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildPodMonitor(t *testing.T) {
	var (
		err error
		o   *kibanacrd.Kibana
		pms []monitoringv1.PodMonitor
	)

	sch := scheme.Scheme
	if err := monitoringv1.AddToScheme(sch); err != nil {
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
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	assert.Empty(t, pms)

	// When prometheus monitoring is disabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: false,
				},
			},
		},
	}
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	assert.Empty(t, pms)

	// When prometheus monitoring is enabled
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	pms, err = buildPodMonitors(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*monitoringv1.PodMonitor](t, "testdata/podmonitor.yml", &pms[0], sch)
}
