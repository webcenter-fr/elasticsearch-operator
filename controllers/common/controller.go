package common

import (
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// PhaseName is the the current phase name (step) on controller
type PhaseName string

func (o PhaseName) String() string {
	return string(o)
}

// Condition is the condition name
type ConditionName string

func (o ConditionName) String() string {
	return string(o)
}

type Controller struct {
	recorder   record.EventRecorder
	log        *logrus.Entry
	reconciler controller.K8sReconciler
}

const (
	ReadyCondition = "Ready"
	ShortenError   = 100
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

func (r *Controller) SetLogger(log *logrus.Entry) {
	r.log = log
}

func (r *Controller) GetLogger() *logrus.Entry {
	return r.log
}

func (r *Controller) SetRecorder(recorder record.EventRecorder) {
	if recorder == nil {
		panic("Recorder can't be nil")
	}
	r.recorder = recorder
}

func (r *Controller) GetRecorder() record.EventRecorder {
	return r.recorder
}

func (r *Controller) SetReconciler(reconciler controller.K8sReconciler) {
	r.reconciler = reconciler
}

func (r *Controller) GetReconciler() controller.K8sReconciler {
	return r.reconciler
}
