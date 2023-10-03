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

package cerebro

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	name      string               = "cerebro"
	finalizer shared.FinalizerName = "cerebro.k8s.webcenter.fr/finalizer"
)

// CerebroReconciler reconciles a Cerebro objectFHost
type CerebroReconciler struct {
	controller.Controller
	controller.MultiPhaseReconcilerAction
	controller.MultiPhaseReconciler
	controller.BaseReconciler
	name string
}

func NewCerebroReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseReconciler controller.Controller) {

	multiPhaseReconciler = &CerebroReconciler{
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
			finalizer,
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

//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes/finalizers,verbs=update
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=hosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=hosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=hosts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cerebro object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *CerebroReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cb := &cerebrocrd.Cerebro{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		cb,
		data,
		r,
		newApplicationSecretReconciler(
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
		newDeploymentReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newIngressReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newLoadBalancerReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *CerebroReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cerebrocrd.Cerebro{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&appv1.Deployment{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&cerebrocrd.Host{}, handler.EnqueueRequestsFromMapFunc(watchHost(h.Client))).
		Complete(h)
}

func (h *CerebroReconciler) Delete(ctx context.Context, o object.MultiPhaseObject, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data)
}

func (h *CerebroReconciler) OnError(ctx context.Context, o object.MultiPhaseObject, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr)
}

func (h *CerebroReconciler) OnSuccess(ctx context.Context, r object.MultiPhaseObject, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*cerebrocrd.Cerebro)

	// Check adeployment is ready
	isReady := true
	dpl := &appv1.Deployment{}
	if err = h.Client.Get(ctx, types.NamespacedName{Name: GetDeploymentName(o), Namespace: o.Namespace}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read Kibana deployment")
		}

		isReady = false
	} else {
		if dpl.Status.ReadyReplicas != o.Spec.Deployment.Replicas {
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

	url, err := h.computeCerebroUrl(ctx, o)
	if err != nil {
		return res, err
	}
	o.Status.Url = url

	return res, nil
}

// computeCerebroUrl permit to get the public cerebro url to put it on status
func (h *CerebroReconciler) computeCerebroUrl(ctx context.Context, cb *cerebrocrd.Cerebro) (target string, err error) {
	var (
		scheme string
		url    string
	)

	if cb.IsIngressEnabled() {
		url = cb.Spec.Endpoint.Ingress.Host

		if cb.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if cb.IsLoadBalancerEnabled() {
		// Need to get lb service to get IP and port
		service := &corev1.Service{}
		if err = h.Client.Get(ctx, types.NamespacedName{Namespace: cb.Namespace, Name: GetLoadBalancerName(cb)}, service); err != nil {
			return "", errors.Wrap(err, "Error when get Load balancer")
		}

		url = fmt.Sprintf("%s:9000", service.Spec.LoadBalancerIP)
		scheme = "http"
	} else {
		url = fmt.Sprintf("%s.%s.svc:9000", GetServiceName(cb), cb.Namespace)
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, url), nil
}
