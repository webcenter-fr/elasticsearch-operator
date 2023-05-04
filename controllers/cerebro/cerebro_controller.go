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

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	CerebroFinalizer     = "cerebro.k8s.webcenter.fr/finalizer"
	CerebroCondition     = "CerebroReady"
	CerebroPhaseRunning  = "running"
	CerebroPhaseStarting = "starting"
)

// CerebroReconciler reconciles a Cerebro object
type CerebroReconciler struct {
	common.Controller
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewCerebroReconciler(client client.Client, scheme *runtime.Scheme) *CerebroReconciler {

	r := &CerebroReconciler{
		Client: client,
		Scheme: scheme,
		name:   "cerebro",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cerebro.k8s.webcenter.fr,resources=cerebroes/finalizers,verbs=update
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
	reconciler, err := controller.NewStdK8sReconciler(r.Client, CerebroFinalizer, r.GetReconciler(), r.GetLogger(), r.GetRecorder())
	if err != nil {
		return ctrl.Result{}, err
	}

	cb := &cerebrocrd.Cerebro{}
	data := map[string]any{}

	applicationSecretReconciler := NewApplicationSecretReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	configMapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	deploymentReconciler := NewDeploymentReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	ingressReconciler := NewIngressReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	loadBalancerReconciler := NewLoadBalancerReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())

	return reconciler.Reconcile(ctx, req, cb, data,
		applicationSecretReconciler,
		configMapReconciler,
		serviceReconciler,
		deploymentReconciler,
		ingressReconciler,
		loadBalancerReconciler,
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
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&source.Kind{Type: &cerebrocrd.Host{}}, handler.EnqueueRequestsFromMapFunc(watchHost(h.Client))).
		Complete(h)
}

// watchHost permit to update configmap to add Elasticsearch cluster on list of know cluster
func watchHost(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listHosts *cerebrocrd.HostList
			fs        fields.Selector
		)

		o := a.(*cerebrocrd.Host)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listHosts = &cerebrocrd.HostList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.cerebroRef.fullname=%s/%s", o.Spec.CerebroRef.Namespace, o.Spec.CerebroRef.Name))
		if err := c.List(context.Background(), listHosts, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listHosts.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Spec.CerebroRef.Name, Namespace: k.Spec.CerebroRef.Namespace}})
		}

		return reconcileRequests
	}
}

// watchConfigMap permit to update if configMapRef change
func watchConfigMap(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listCerebros *cerebrocrd.CerebroList
			fs           fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Env of type configMap
		listCerebros = &cerebrocrd.CerebroList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.configMapKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listCerebros, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listCerebros.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type configMap
		listCerebros = &cerebrocrd.CerebroList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.configMapRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listCerebros, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listCerebros.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests

	}
}

// watchSecret permit to update Kibana if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listCerebros *cerebrocrd.CerebroList
			fs           fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Env of type secrets
		listCerebros = &cerebrocrd.CerebroList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.secretKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listCerebros, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listCerebros.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listCerebros = &cerebrocrd.CerebroList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listCerebros, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listCerebros.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

func (h *CerebroReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)

	o.Status.IsError = pointer.Bool(false)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, CerebroCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   CerebroCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	return res, nil
}
func (h *CerebroReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *CerebroReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *CerebroReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := r.(*cerebrocrd.Cerebro)

	o.Status.IsError = pointer.Bool(true)

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    CerebroCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	common.TotalErrors.Inc()
	return res, currentErr
}
func (h *CerebroReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
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
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, CerebroCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   CerebroCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		if o.Status.Phase != CerebroPhaseRunning {
			o.Status.Phase = CerebroPhaseRunning
		}

		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   common.ReadyCondition,
				Reason: "Available",
				Status: metav1.ConditionTrue,
			})
		}

	} else {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, CerebroCondition, metav1.ConditionTrue) || (condition.FindStatusCondition(o.Status.Conditions, CerebroCondition).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   CerebroCondition,
				Status: metav1.ConditionFalse,
				Reason: "NotReady",
			})
		}

		if o.Status.Phase != CerebroPhaseStarting {
			o.Status.Phase = CerebroPhaseStarting
		}

		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue) || (condition.FindStatusCondition(o.Status.Conditions, common.ReadyCondition).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   common.ReadyCondition,
				Reason: "NotReady",
				Status: metav1.ConditionFalse,
			})
		}

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

func (h *CerebroReconciler) Name() string {
	return "cerebro"
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
