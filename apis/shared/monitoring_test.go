package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPrometheusMonitoring(t *testing.T) {
	var o MonitoringSpec

	// With default values
	o = MonitoringSpec{}
	assert.False(t, o.IsPrometheusMonitoring())

	// When enabled
	o = MonitoringSpec{
		Prometheus: &MonitoringPrometheusSpec{
			Enabled: true,
		},
	}
	assert.True(t, o.IsPrometheusMonitoring())

	// When disabled
	o = MonitoringSpec{
		Prometheus: &MonitoringPrometheusSpec{
			Enabled: false,
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
			Enabled: true,
		},
	}
	assert.True(t, o.IsMetricbeatMonitoring(1))

	// When disabled
	o = MonitoringSpec{
		Metricbeat: &MonitoringMetricbeatSpec{
			Enabled: false,
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring(1))

	// When enabled but replica is set to 0
	o = MonitoringSpec{
		Metricbeat: &MonitoringMetricbeatSpec{
			Enabled: true,
		},
	}
	assert.False(t, o.IsMetricbeatMonitoring(0))
}
