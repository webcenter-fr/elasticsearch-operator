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
	"fmt"
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
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
	MetricbeatFinalizer     = "metricbeat.k8s.webcenter.fr/finalizer"
	MetricbeatCondition     = "metricbeatReady"
	MetricbeatPhaseRunning  = "running"
	MetricbeatPhaseStarting = "starting"
)

// MetricbeatReconciler reconciles a Metricbeat object
type MetricbeatReconciler struct {
	common.Controller
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewMetricbeatReconciler(client client.Client, scheme *runtime.Scheme) *MetricbeatReconciler {

	r := &MetricbeatReconciler{
		Client: client,
		Scheme: scheme,
		name:   "metricbeat",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
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
//+kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
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
	reconciler, err := controller.NewStdK8sReconciler(r.Client, MetricbeatFinalizer, r.GetReconciler(), r.GetLogger(), r.GetRecorder())
	if err != nil {
		return ctrl.Result{}, err
	}

	mb := &beatcrd.Metricbeat{}
	data := map[string]any{}

	caElasticsearchReconciler := NewCAElasticsearchReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	credentialReconciler := NewCredentialReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	configMapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	pdbReconciler := NewPdbReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	networkPolicyReconciler := NewNetworkPolicyReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	statefulsetReconciler := NewStatefulsetReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())

	return reconciler.Reconcile(ctx, req, mb, data,
		caElasticsearchReconciler,
		credentialReconciler,
		configMapReconciler,
		serviceReconciler,
		pdbReconciler,
		networkPolicyReconciler,
		statefulsetReconciler,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *MetricbeatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&beatcrd.Metricbeat{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&source.Kind{Type: &elasticsearchcrd.Elasticsearch{}}, handler.EnqueueRequestsFromMapFunc(watchElasticsearch(h.Client))).
		Complete(h)
}

// watchElasticsearch permit to update if ElasticsearchRef change
func watchElasticsearch(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests

	}
}

// watchConfigMap permit to update if configMapRef change
func watchConfigMap(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Additional volumes secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.additionalVolumes.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Env of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests

	}
}

// watchSecret permit to update Metricbeat if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Elasticsearch API cert secret when external
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.elasticsearchCASecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch credentials when external
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.external.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Additional volumes secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.additionalVolumes.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Env of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

func (h *MetricbeatReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)

	o.Status.IsError = pointer.Bool(false)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, MetricbeatCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   MetricbeatCondition,
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
func (h *MetricbeatReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *MetricbeatReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *MetricbeatReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := r.(*beatcrd.Metricbeat)

	o.Status.IsError = pointer.Bool(true)

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    MetricbeatCondition,
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
func (h *MetricbeatReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*beatcrd.Metricbeat)

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
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, MetricbeatCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   MetricbeatCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		if o.Status.Phase != MetricbeatPhaseRunning {
			o.Status.Phase = MetricbeatPhaseRunning
		}

		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   common.ReadyCondition,
				Reason: "Available",
				Status: metav1.ConditionTrue,
			})
		}

	} else {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, MetricbeatCondition, metav1.ConditionTrue) || (condition.FindStatusCondition(o.Status.Conditions, MetricbeatCondition).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   MetricbeatCondition,
				Status: metav1.ConditionFalse,
				Reason: "NotReady",
			})
		}

		if o.Status.Phase != MetricbeatPhaseStarting {
			o.Status.Phase = MetricbeatPhaseStarting
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

	return res, nil
}

func (h *MetricbeatReconciler) Name() string {
	return "metricbeat"
}
