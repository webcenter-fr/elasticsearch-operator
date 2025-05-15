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

package filebeat

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	name              string               = "filebeat"
	filebeatFinalizer shared.FinalizerName = "filebeat.k8s.webcenter.fr/finalizer"
)

// FilebeatReconciler reconciles a Filebeat object
type FilebeatReconciler struct {
	controller.Controller
	multiphase.MultiPhaseReconciler[*beatcrd.Filebeat]
	multiphase.MultiPhaseReconcilerAction[*beatcrd.Filebeat]
	name            string
	stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, client.Object]
	kubeCapability  common.KubernetesCapability
}

func NewFilebeatReconciler(c client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &FilebeatReconciler{
		Controller: controller.NewController(),
		MultiPhaseReconciler: multiphase.NewMultiPhaseReconciler[*beatcrd.Filebeat](
			c,
			name,
			filebeatFinalizer,
			logger,
			recorder,
		),
		MultiPhaseReconcilerAction: multiphase.NewMultiPhaseReconcilerAction[*beatcrd.Filebeat](
			c,
			controller.ReadyCondition,
			recorder,
		),

		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, client.Object]{
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ServiceAccount, client.Object](newServiceAccountReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *rbacv1.RoleBinding, client.Object](newRoleBindingReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret, client.Object](newTlsReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret, client.Object](newCALogstashReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret, client.Object](newCAElasticsearchReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret, client.Object](newCredentialReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ConfigMap, client.Object](newConfiMapReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Service, client.Object](newServiceReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *policyv1.PodDisruptionBudget, client.Object](newPdbReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *appv1.StatefulSet, client.Object](newStatefulsetReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *networkingv1.Ingress, client.Object](newIngressReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *beatcrd.Metricbeat, client.Object](newMetricbeatReconciler(c, recorder)),
		},
	}

	// Add route reconciler if CRD exist on cluster
	if kubeCapability.HasRoute {
		reconciler.stepReconcilers = append(reconciler.stepReconcilers, multiphase.NewObjectMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *routev1.Route, client.Object](newRouteReconciler(c, recorder)))
	}

	return reconciler
}

//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=filebeats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=filebeats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=beat.k8s.webcenter.fr,resources=filebeats/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="beat.k8s.webcenter.fr",resources=metricbeats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="route.openshift.io",resources=routes/custom-host,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources=CustomResourceDefinition,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="security.openshift.io",resources=securitycontextconstraints,verbs=use

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Filebeat object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *FilebeatReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	fb := &beatcrd.Filebeat{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		fb,
		data,
		r,
		r.stepReconcilers...,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *FilebeatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&beatcrd.Filebeat{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&beatcrd.Metricbeat{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client()))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearch(h.Client()))).
		Watches(&logstashcrd.Logstash{}, handler.EnqueueRequestsFromMapFunc(watchLogstash(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		})

	if h.kubeCapability.HasRoute {
		ctrlBuilder.Owns(&routev1.Route{})
	}

	return ctrlBuilder.Complete(h)
}

func (h *FilebeatReconciler) Client() client.Client {
	return h.MultiPhaseReconcilerAction.Client()
}

func (h *FilebeatReconciler) Recorder() record.EventRecorder {
	return h.MultiPhaseReconcilerAction.Recorder()
}

func (h *FilebeatReconciler) Configure(ctx context.Context, req reconcile.Request, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(1)

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o, data, logger)
}

func (h *FilebeatReconciler) Delete(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *FilebeatReconciler) OnError(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, currentErr error, logger *logrus.Entry) (res reconcile.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *FilebeatReconciler) OnSuccess(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {

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

	// Check statefulset is ready
	isReady := true
	sts := &appv1.StatefulSet{}
	if err = h.Client().Get(ctx, types.NamespacedName{Name: GetStatefulsetName(o), Namespace: o.Namespace}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read Filebeat statefulset")
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

	o.Status.CertSecretName = GetSecretNameForTls(o)

	return res, nil
}
