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
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

const (
	name string = "kibana"
)

// KibanaReconciler reconciles a Kibana object
type KibanaReconciler struct {
	controller.Controller
	controller.MultiPhaseReconcilerAction
	controller.MultiPhaseReconciler
	controller.BaseReconciler
	name string
}

func NewKibanaReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseReconciler controller.Controller) {

	multiPhaseReconciler = &KibanaReconciler{
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
			"kibana.k8s.webcenter.fr/finalizer",
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
	kb := &kibanacrd.Kibana{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		kb,
		data,
		r,
		newTlsReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
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
		newNetworkPolicyReconciler(
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
		newPodMonitorReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newMetricbeatReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
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

func (h *KibanaReconciler) Delete(ctx context.Context, o object.MultiPhaseObject, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data)
}

func (h *KibanaReconciler) OnError(ctx context.Context, o object.MultiPhaseObject, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr)
}

func (h *KibanaReconciler) OnSuccess(ctx context.Context, r object.MultiPhaseObject, data map[string]any) (res ctrl.Result, err error) {
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

	url, err := h.computeKibanaUrl(ctx, o)
	if err != nil {
		return res, err
	}
	o.Status.Url = url

	return res, nil
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
