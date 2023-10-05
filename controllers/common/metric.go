package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TotalErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_errors",
			Help: "Number of errors from all controllers",
		},
	)
	ControllerMetrics = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_total",
		Help: "Total number of resource handled per controller",
	}, []string{"controller"})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(TotalErrors, ControllerMetrics)
}
