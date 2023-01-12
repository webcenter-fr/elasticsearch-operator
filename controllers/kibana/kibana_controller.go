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
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
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
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

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

	tlsReconciler := NewTlsReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "tls",
		}),
	})

	caElasticsearchReconciler := NewCAElasticsearchReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "caElasticsearch",
		}),
	})

	credentialReconciler := NewCredentialReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "credential",
		}),
	})

	configMapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "configMap",
		}),
	})

	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "service",
		}),
	})

	pdbReconciler := NewPdbReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "pdb",
		}),
	})

	deploymentReconciler := NewDeploymentReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "deployment",
		}),
	})

	ingressReconciler := NewIngressReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "ingress",
		}),
	})

	loadBalancerReconciler := NewLoadBalancerReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "loadBalancer",
		}),
	})

	return reconciler.Reconcile(ctx, req, kb, data,
		tlsReconciler,
		caElasticsearchReconciler,
		credentialReconciler,
		configMapReconciler,
		serviceReconciler,
		pdbReconciler,
		deploymentReconciler,
		ingressReconciler,
		loadBalancerReconciler,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *KibanaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanacrd.Kibana{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Complete(h)
}

// watchSecret permit to update Kibana if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
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

		// Elasticsearch API secret
		if elasticsearchcontrollers.GetElasticsearchNameFromSecretApiTlsName(a.GetName()) != "" {
			listKibanas = &kibanacrd.KibanaList{}
			fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.name=%s", elasticsearchcontrollers.GetElasticsearchNameFromSecretApiTlsName(a.GetName())))
			// Get all kibana linked with secret
			if err := c.List(context.Background(), listKibanas, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
				panic(err)
			}
			for _, k := range listKibanas.Items {
				reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
			}
		}

		return reconcileRequests
	}
}

func (h *KibanaReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, KibanaCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   KibanaCondition,
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
	common.TotalErrors.Inc()
	return res, currentErr
}
func (h *KibanaReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*kibanacrd.Kibana)

	// Wait few time, to be sure Satefulset created
	time.Sleep(1 * time.Second)

	// Check adeployment is ready
	dpl := &appv1.Deployment{}
	if err = h.Client.Get(ctx, types.NamespacedName{Name: GetDeploymentName(o), Namespace: o.Namespace}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read Kibana deployment")
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil

	}

	if dpl.Status.ReadyReplicas == *dpl.Spec.Replicas {
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, KibanaCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   KibanaCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})

			o.Status.Phase = KibanaPhaseRunning
		}

		return res, nil
	}

	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, KibanaCondition, metav1.ConditionFalse) || (condition.FindStatusCondition(o.Status.Conditions, KibanaCondition) != nil && condition.FindStatusCondition(o.Status.Conditions, KibanaCondition).Reason != "NotReady") {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   KibanaCondition,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})

	}

	o.Status.Phase = KibanaPhaseStarting

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (h *KibanaReconciler) Name() string {
	return "kibana"
}
