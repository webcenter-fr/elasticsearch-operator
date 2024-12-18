package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TotalErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "elasticsearch_operator_errors_total",
			Help: "Number of errors from all controllers",
		},
	)
	ControllerErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "elasticsearch_operator_errors_controller",
		Help: "Number of errors per controllers",
	}, []string{"controller", "namespace", "name"})
	ControllerInstances = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "elasticsearch_operator_instances_controller",
		Help: "Number of instance per controllers",
	}, []string{"controller", "namespace", "name"})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(TotalErrors, ControllerErrors, ControllerInstances)
}
