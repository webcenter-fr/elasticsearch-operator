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
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
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
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	name            string
	kubeCapability  common.KubernetesCapability
	stepReconcilers []controller.MultiPhaseStepReconcilerAction
}

func NewCerebroReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &CerebroReconciler{
		Controller: controller.NewBasicController(),
		MultiPhaseReconcilerAction: controller.NewBasicMultiPhaseReconcilerAction(
			client,
			controller.ReadyCondition,
			recorder,
		),
		MultiPhaseReconciler: controller.NewBasicMultiPhaseReconciler(
			client,
			name,
			finalizer,
			logger,
			recorder,
		),
		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []controller.MultiPhaseStepReconcilerAction{
			newApplicationSecretReconciler(client, recorder),
			newConfiMapReconciler(client, recorder),
			newServiceReconciler(client, recorder),
			newDeploymentReconciler(client, recorder, kubeCapability.HasRoute),
			newIngressReconciler(client, recorder),
			newLoadBalancerReconciler(client, recorder),
		},
	}

	if kubeCapability.HasRoute {
		reconciler.stepReconcilers = append(reconciler.stepReconcilers, newRouteReconciler(client, recorder))
	}

	return reconciler
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
//+kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="route.openshift.io",resources=routes/custom-host,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources=CustomResourceDefinition,verbs=get;list;watch

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
		r.stepReconcilers...,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *CerebroReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&cerebrocrd.Cerebro{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&appv1.Deployment{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client()))).
		Watches(&cerebrocrd.Host{}, handler.EnqueueRequestsFromMapFunc(watchHost(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: common.DefaultControllerRateLimiter[reconcile.Request](),
		})

	if h.kubeCapability.HasRoute {
		ctrlBuilder.Owns(&routev1.Route{})
	}

	return ctrlBuilder.Complete(h)
}

func (h *CerebroReconciler) Configure(ctx context.Context, req ctrl.Request, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (res ctrl.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, resource.GetNamespace(), resource.GetName()).Set(1)

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, resource, data, logger)
}

func (h *CerebroReconciler) Delete(ctx context.Context, o object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *CerebroReconciler) OnError(ctx context.Context, o object.MultiPhaseObject, data map[string]any, currentErr error, logger *logrus.Entry) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *CerebroReconciler) OnSuccess(ctx context.Context, r object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (res ctrl.Result, err error) {
	o := r.(*cerebrocrd.Cerebro)

	// Reset the current cluster errors
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	// Not preserve condition to avoid to update status each time
	conditions := o.GetStatus().GetConditions()
	o.GetStatus().SetConditions(nil)
	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data, logger)
	if err != nil {
		return res, err
	}
	o.GetStatus().SetConditions(conditions)

	// Check adeployment is ready
	isReady := true
	dpl := &appv1.Deployment{}
	if err = h.Client().Get(ctx, types.NamespacedName{Name: GetDeploymentName(o), Namespace: o.Namespace}, dpl); err != nil {
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

	if cb.Spec.Endpoint.IsIngressEnabled() {
		url = cb.Spec.Endpoint.Ingress.Host

		if cb.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if cb.Spec.Endpoint.IsRouteEnabled() {
		url = cb.Spec.Endpoint.Route.Host

		if cb.Spec.Endpoint.Route.TlsEnabled != nil && *cb.Spec.Endpoint.Route.TlsEnabled {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if cb.Spec.Endpoint.IsLoadBalancerEnabled() {
		// Need to get lb service to get IP and port
		service := &corev1.Service{}
		if err = h.Client().Get(ctx, types.NamespacedName{Namespace: cb.Namespace, Name: GetLoadBalancerName(cb)}, service); err != nil {
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
