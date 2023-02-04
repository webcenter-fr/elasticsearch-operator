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

package logstash

import (
	"context"
	"fmt"
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	LogstashFinalizer     = "logstash.k8s.webcenter.fr/finalizer"
	LogstashCondition     = "logstashReady"
	LogstashPhaseRunning  = "running"
	LogstashPhaseStarting = "starting"
)

// LogstashReconciler reconciles a Logstash object
type LogstashReconciler struct {
	common.Controller
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewLogstashReconciler(client client.Client, scheme *runtime.Scheme) *LogstashReconciler {

	r := &LogstashReconciler{
		Client: client,
		Scheme: scheme,
		name:   "kibana",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=logstash.k8s.webcenter.fr,resources=logstashes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=logstash.k8s.webcenter.fr,resources=logstashes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=logstash.k8s.webcenter.fr,resources=logstashes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Logstash object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *LogstashReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconciler, err := controller.NewStdK8sReconciler(r.Client, LogstashFinalizer, r.GetReconciler(), r.GetLogger(), r.GetRecorder())
	if err != nil {
		return ctrl.Result{}, err
	}

	ls := &logstashcrd.Logstash{}
	data := map[string]any{}

	caElasticsearchReconciler := NewCAElasticsearchReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	credentialReconciler := NewCredentialReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	configMapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	pdbReconciler := NewPdbReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	ingressReconciler := NewIngressReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	networkPolicyReconciler := NewNetworkPolicyReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())
	statefulsetReconciler := NewStatefulsetReconciler(r.Client, r.Scheme, r.GetRecorder(), r.GetLogger())

	return reconciler.Reconcile(ctx, req, ls, data,
		caElasticsearchReconciler,
		credentialReconciler,
		configMapReconciler,
		serviceReconciler,
		pdbReconciler,
		networkPolicyReconciler,
		statefulsetReconciler,
		ingressReconciler,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *LogstashReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&logstashcrd.Logstash{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Complete(h)
}

// watchSecret permit to update Logstash if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listLogstashs *logstashcrd.LogstashList
			fs            fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Keystore secret
		listLogstashs = &logstashcrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.keystoreSecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch API cert secret when managed
		if elasticsearchcontrollers.GetElasticsearchNameFromSecretApiTlsName(a.GetName()) != "" {
			listLogstashs = &logstashcrd.LogstashList{}
			fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.name=%s", elasticsearchcontrollers.GetElasticsearchNameFromSecretApiTlsName(a.GetName())))
			if err := c.List(context.Background(), listLogstashs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
				panic(err)
			}
			for _, k := range listLogstashs.Items {
				reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
			}
		}

		// Elasticsearch API cert secret when external
		listLogstashs = &logstashcrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.elasticsearchCASecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch credentials when external
		listLogstashs = &logstashcrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.external.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

func (h *LogstashReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, LogstashCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   LogstashCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	return res, nil
}
func (h *LogstashReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *LogstashReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *LogstashReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return res, currentErr
}
func (h *LogstashReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*logstashcrd.Logstash)

	// Check statefulset is ready
	isReady := true
	sts := &appv1.StatefulSet{}
	if err = h.Client.Get(ctx, types.NamespacedName{Name: GetStatefulsetName(o), Namespace: o.Namespace}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read Logstash statefulset")
		}

		isReady = false
	} else {
		if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
			isReady = false
		}
	}

	if isReady {
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, LogstashCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   LogstashCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		if o.Status.Phase != LogstashPhaseRunning {
			o.Status.Phase = LogstashPhaseRunning
		}

	} else {
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, LogstashCondition, metav1.ConditionFalse) || (condition.FindStatusCondition(o.Status.Conditions, LogstashCondition) != nil && condition.FindStatusCondition(o.Status.Conditions, LogstashCondition).Reason != "NotReady") {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   LogstashCondition,
				Status: metav1.ConditionFalse,
				Reason: "NotReady",
			})
		}

		if o.Status.Phase != LogstashPhaseStarting {
			o.Status.Phase = LogstashPhaseStarting
		}

		// Requeued to check if status change
		res.RequeueAfter = time.Second * 30
	}

	return res, nil
}

func (h *LogstashReconciler) Name() string {
	return "logstash"
}