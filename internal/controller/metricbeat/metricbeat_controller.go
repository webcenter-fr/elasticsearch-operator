/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricbeat

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

const (
	name string = "metricbeat"
)

// MetricbeatReconciler reconciles a Metricbeat object
type MetricbeatReconciler struct {
	controller.Controller
	controller.MultiPhaseReconcilerAction
	controller.MultiPhaseReconciler
	controller.BaseReconciler
	name string
}

func NewMetricbeatReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseReconciler controller.Controller) {

	multiPhaseReconciler = &MetricbeatReconciler{
		Controller: controller.NewBasicController(),
		MultiPhaseReconcilerAction: controller.NewBasicMultiPhaseReconcilerAction(
			client,
			controller.ReadyCondition,
			logger,
			recorder,
		),
		MultiPhaseReconciler: controller.NewBasicMultiPhaseReconciler(
			client,
			name,
			"metricbeat.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
		name: name,
	}

	common.ControllerMetrics.WithLabelValues(name).Add(0)

	return multiPhaseReconciler
}

//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=metricbeats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=metricbeats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=metricbeats/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the metricbeat object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MetricbeatReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mb := &beatcrd.Metricbeat{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		mb,
		data,
		r,
		newCAElasticsearchReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newCredentialReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newConfiMapReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newServiceReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newPdbReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newStatefulsetReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *MetricbeatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&beatcrd.Metricbeat{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearch(h.Client))).
		Complete(h)
}

func (h *MetricbeatReconciler) Delete(ctx context.Context, o object.MultiPhaseObject, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data)
}

func (h *MetricbeatReconciler) OnError(ctx context.Context, o object.MultiPhaseObject, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr)
}

func (h *MetricbeatReconciler) OnSuccess(ctx context.Context, r object.MultiPhaseObject, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*beatcrd.Metricbeat)

	// Not preserve condition to avoid to update status each time
	conditions := o.GetStatus().GetConditions()
	o.GetStatus().SetConditions(nil)
	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data)
	if err != nil {
		return res, err
	}
	o.GetStatus().SetConditions(conditions)

	// Check statefulset is ready
	isReady := true
	sts := &appv1.StatefulSet{}
	if err = h.Client.Get(ctx, types.NamespacedName{Name: GetStatefulsetName(o), Namespace: o.Namespace}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read metricbeat statefulset")
		}

		isReady = false
	} else {
		if sts.Status.ReadyReplicas != o.Spec.Deployment.Replicas {
			isReady = false
		}
	}

	if isReady {
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   controller.ReadyCondition.String(),
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		o.Status.PhaseName = controller.RunningPhase

	} else {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue) || (condition.FindStatusCondition(o.Status.Conditions, controller.ReadyCondition.String()).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   controller.ReadyCondition.String(),
				Status: metav1.ConditionFalse,
				Reason: "NotReady",
			})
		}

		o.Status.PhaseName = controller.StartingPhase

		// Requeued to check if status change
		res.RequeueAfter = time.Second * 30
	}

	return res, nil
}
