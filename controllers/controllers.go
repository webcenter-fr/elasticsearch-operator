package controllers

import (
	"fmt"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Reconciler struct {
	recorder   record.EventRecorder
	log        *logrus.Entry
	reconciler controller.K8sReconciler
}

type CompareResource struct {
	Current  client.Object
	Expected client.Object
	Diff     *controller.K8sDiff
}

var (
	totalErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_errors",
			Help: "Number of errors from all controllers",
		},
	)
	controllerMetrics = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_total",
		Help: "Total number of resource handled per controller",
	}, []string{"controller"})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(totalErrors, controllerMetrics)
}

func (r *Reconciler) SetLogger(log *logrus.Entry) {
	r.log = log
}

func (r *Reconciler) SetRecorder(recorder record.EventRecorder) {
	if recorder == nil {
		panic("Recorder can't be nil")
	}
	r.recorder = recorder
}

func (r *Reconciler) SetReconsiler(reconciler controller.K8sReconciler) {
	r.reconciler = reconciler
}

func GetNodeName(clusterName, groupName string, index int) string {
	return fmt.Sprintf("%s-%s-%d", clusterName, groupName, index)
}
