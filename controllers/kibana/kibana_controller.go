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

package kibana

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
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
)

const (
	KibanaFinalizer     = "kibana.k8s.webcenter.fr/finalizer"
	KibanaCondition     = "KibanaReady"
	KibanaPhaseRunning  = "running"
	KibanaPhaseStarting = "starting"
)

// KibanaReconciler reconciles a Kibana object
type KibanaReconciler struct {
	common.Controller
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewKibanaReconciler(client client.Client, scheme *runtime.Scheme) *KibanaReconciler {

	r := &KibanaReconciler{
		Client: client,
		Scheme: scheme,
		name:   "kibana",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=kibana.k8s.webcenter.fr,resources=kibanas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibana.k8s.webcenter.fr,resources=kibanas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibana.k8s.webcenter.fr,resources=kibanas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="monitoring.coreos.com",resources=podmonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="beat.k8s.webcenter.fr",resources=metricbeats,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kibana object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *KibanaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconciler, err := controller.NewStdK8sReconciler(r.Client, KibanaFinalizer, r.GetReconciler(), r.GetLogger(), r.GetRecorder())
	if err != nil {
		return ctrl.Result{}, err
	}

	kb := &kibanacrd.Kibana{}
	data := map[string]any{}

	tlsReconciler := NewTlsReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	caElasticsearchReconciler := NewCAElasticsearchReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	credentialReconciler := NewCredentialReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	configMapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	pdbReconciler := NewPdbReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	deploymentReconciler := NewDeploymentReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	ingressReconciler := NewIngressReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	loadBalancerReconciler := NewLoadBalancerReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	networkPolicyReconciler := NewNetworkPolicyReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	podMonitorReconciler := NewPodMonitorReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	metricbeatReconsiler := NewMetricbeatReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())

	return reconciler.Reconcile(ctx, req, kb, data,
		tlsReconciler,
		caElasticsearchReconciler,
		credentialReconciler,
		configMapReconciler,
		serviceReconciler,
		pdbReconciler,
		networkPolicyReconciler,
		deploymentReconciler,
		ingressReconciler,
		loadBalancerReconciler,
		podMonitorReconciler,
		metricbeatReconsiler,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *KibanaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanacrd.Kibana{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.Deployment{}).
		Owns(&beatcrd.Metricbeat{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearch(h.Client))).
		Complete(h)
}

// watchElasticsearch permit to update if ElasticsearchRef change
func watchElasticsearch(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKibanas *kibanacrd.KibanaList
			fs          fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests

	}
}

// watchConfigMap permit to update if configMapRef change
func watchConfigMap(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKibanas *kibanacrd.KibanaList
			fs          fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Env of type configMap
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.configMapKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type configMap
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.configMapRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests

	}
}

// watchSecret permit to update Kibana if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKibanas *kibanacrd.KibanaList
			fs          fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Keystore secret
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.keystoreSecretRef.name=%s", a.GetName()))
		// Get all kibana linked with secret
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// external TLS secret
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.tls.certificateSecretRef.name=%s", a.GetName()))
		// Get all kibana linked with secret
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch API cert secret when external
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.elasticsearchCASecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch credentials when external
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.external.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Env of type secrets
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.secretKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listKibanas = &kibanacrd.KibanaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKibanas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

func (h *KibanaReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	o.Status.IsError = pointer.Bool(false)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, KibanaCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   KibanaCondition,
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
func (h *KibanaReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *KibanaReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *KibanaReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := r.(*kibanacrd.Kibana)

	o.Status.IsError = pointer.Bool(true)

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    KibanaCondition,
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
func (h *KibanaReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*kibanacrd.Kibana)

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
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, KibanaCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   KibanaCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		if o.Status.Phase != KibanaPhaseRunning {
			o.Status.Phase = KibanaPhaseRunning
		}

		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   common.ReadyCondition,
				Reason: "Available",
				Status: metav1.ConditionTrue,
			})
		}

	} else {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, KibanaCondition, metav1.ConditionTrue) || (condition.FindStatusCondition(o.Status.Conditions, KibanaCondition).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   KibanaCondition,
				Status: metav1.ConditionFalse,
				Reason: "NotReady",
			})
		}

		if o.Status.Phase != KibanaPhaseStarting {
			o.Status.Phase = KibanaPhaseStarting
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

	url, err := h.computeKibanaUrl(ctx, o)
	if err != nil {
		return res, err
	}
	o.Status.Url = url

	return res, nil
}

func (h *KibanaReconciler) Name() string {
	return "kibana"
}

// computeKibanaUrl permit to get the public Kibana url to put it on status
func (h *KibanaReconciler) computeKibanaUrl(ctx context.Context, kb *kibanacrd.Kibana) (target string, err error) {
	var (
		scheme string
		url    string
	)

	if kb.IsIngressEnabled() {
		url = kb.Spec.Endpoint.Ingress.Host

		if kb.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if kb.IsLoadBalancerEnabled() {
		// Need to get lb service to get IP and port
		service := &corev1.Service{}
		if err = h.Client.Get(ctx, types.NamespacedName{Namespace: kb.Namespace, Name: GetLoadBalancerName(kb)}, service); err != nil {
			return "", errors.Wrap(err, "Error when get Load balancer")
		}

		url = fmt.Sprintf("%s:9200", service.Spec.LoadBalancerIP)
		if kb.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else {
		url = fmt.Sprintf("%s.%s.svc:9200", GetServiceName(kb), kb.Namespace)
		if kb.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return fmt.Sprintf("%s://%s", scheme, url), nil
}
