package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestIsPrometheusMonitoring(t *testing.T) {
	var o MonitoringSpec

	// With default values
	o = MonitoringSpec{}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = MonitoringSpec{
		Prometheus: &MonitoringPrometheusSpec{
			Enabled: ptr.To[bool](true),
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = MonitoringSpec{
		Prometheus: &MonitoringPrometheusSpec{
			Enabled: ptr.To[bool](false),
		},
	}
	assert.False(t, o.IsPrometheusMonitoring())
}

func TestIsMetricbeatMonitoring(t *testing.T) {
	var o MonitoringSpec

	// With default values
	o = MonitoringSpec{}
	assert.False(t, o.IsMetricbeatMonitoring(0))

	// When enabled
	o = MonitoringSpec{
		Metricbeat: &MonitoringMetricbeatSpec{
			Enabled: ptr.To(true),
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring(1))

	// When disabled
	o = MonitoringSpec{
		Metricbeat: &MonitoringMetricbeatSpec{
			Enabled: ptr.To(false),
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring(1))

	// When enabled but replica is set to 0
	o = MonitoringSpec{
		Metricbeat: &MonitoringMetricbeatSpec{
			Enabled: ptr.To(true),
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring(0))
}
