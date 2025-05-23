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

package elasticsearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elastic "github.com/elastic/go-elasticsearch/v8"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	name                   string               = "elasticsearch"
	elasticsearchFinalizer shared.FinalizerName = "elasticsearch.k8s.webcenter.fr/finalizer"
)

// ElasticsearchReconciler reconciles a Elasticsearch object
type ElasticsearchReconciler struct {
	controller.Controller
	multiphase.MultiPhaseReconciler[*elasticsearchcrd.Elasticsearch]
	multiphase.MultiPhaseReconcilerAction[*elasticsearchcrd.Elasticsearch]
	name            string
	stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, client.Object]
	kubeCapability  common.KubernetesCapability
}

// NewElasticsearchReconciler is the default constructor for Elasticsearch controller
func NewElasticsearchReconciler(c client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &ElasticsearchReconciler{
		Controller: controller.NewController(),
		MultiPhaseReconciler: multiphase.NewMultiPhaseReconciler[*elasticsearchcrd.Elasticsearch](
			c,
			name,
			elasticsearchFinalizer,
			logger,
			recorder,
		),
		MultiPhaseReconcilerAction: multiphase.NewMultiPhaseReconcilerAction[*elasticsearchcrd.Elasticsearch](
			c,
			controller.ReadyCondition,
			recorder,
		),
		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, client.Object]{
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.ServiceAccount, client.Object](newServiceAccountReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *rbacv1.RoleBinding, client.Object](newRoleBindingReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret, client.Object](newTlsReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret, client.Object](newCredentialReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.License, client.Object](newLicenseReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.ConfigMap, client.Object](newConfiMapReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Service, client.Object](newServiceReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *policyv1.PodDisruptionBudget, client.Object](newPdbReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *networkingv1.NetworkPolicy, client.Object](newNetworkPolicyReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet, client.Object](newStatefulsetReconciler(c, recorder, kubeCapability.HasRoute)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.User, client.Object](newSystemUserReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *networkingv1.Ingress, client.Object](newIngressReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Service, client.Object](newLoadBalancerReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *beatcrd.Metricbeat, client.Object](newMetricbeatReconciler(c, recorder)),
			multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.Deployment, client.Object](newExporterReconciler(c, recorder)),
		},
	}

	// Add Pod monitor reconciler is CRD exist on cluster
	if kubeCapability.HasPrometheus {
		reconciler.stepReconcilers = append(reconciler.stepReconcilers, multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *monitoringv1.PodMonitor, client.Object](newPodMonitorReconciler(c, recorder)))
	}

	// Add route reconciler if CRD exist on cluster
	if kubeCapability.HasRoute {
		reconciler.stepReconcilers = append(reconciler.stepReconcilers, multiphase.NewObjectMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *routev1.Route, client.Object](newRouteReconciler(c, recorder)))
	}

	return reconciler
}

//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="monitoring.coreos.com",resources=podmonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="elasticsearchapi.k8s.webcenter.fr",resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="elasticsearchapi.k8s.webcenter.fr",resources=licenses,verbs=get;list;watch;create;update;patch;delete
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
// the Elasticsearch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ElasticsearchReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	es := &elasticsearchcrd.Elasticsearch{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		es,
		data,
		r,
		r.stepReconcilers...,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *ElasticsearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchcrd.Elasticsearch{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Owns(&appv1.StatefulSet{}).
		Owns(&appv1.Deployment{}).
		Owns(&elasticsearchapicrd.User{}).
		Owns(&elasticsearchapicrd.License{}).
		Owns(&beatcrd.Metricbeat{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client()))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearchMonitoring(h.Client()))).
		Watches(&cerebrocrd.Host{}, handler.EnqueueRequestsFromMapFunc(watchHost(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		})

	if h.kubeCapability.HasRoute {
		ctrlBuilder.Owns(&routev1.Route{})
	}

	if h.kubeCapability.HasPrometheus {
		ctrlBuilder.Owns(&monitoringv1.PodMonitor{})
	}

	return ctrlBuilder.Complete(h)
}

func (h *ElasticsearchReconciler) Client() client.Client {
	return h.MultiPhaseReconcilerAction.Client()
}

/*************  ✨ Windsurf Command ⭐  *************/
// Recorder returns the event recorder associated with the ElasticsearchReconciler.
// This recorder is used to generate Kubernetes events for the reconciler's actions.

/*******  9fb4ec25-b46a-4c4e-984b-89dd1f4e34fa  *******/
func (h *ElasticsearchReconciler) Recorder() record.EventRecorder {
	return h.MultiPhaseReconcilerAction.Recorder()
}

func (h *ElasticsearchReconciler) Configure(ctx context.Context, req reconcile.Request, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.Namespace, o.Name).Set(1)

	if o.Status.IsBootstrapping == nil {
		o.Status.IsBootstrapping = ptr.To[bool](false)
	}

	// Get Elasticsearch health
	// Not blocking way, cluster can be unreachable
	esHandler, err := h.getElasticsearchHandler(ctx, o, logger)
	if err != nil {
		logger.Warnf("Error when get elasticsearch client: %s", err.Error())
		o.Status.Health = "Unreachable"
	} else {
		if esHandler == nil {
			o.Status.Health = "Unreachable"
		} else {
			data["esHandler"] = esHandler
			health, err := esHandler.ClusterHealth()
			if err != nil {
				logger.Warnf("Error when get elasticsearch health: %s", err.Error())
				o.Status.Health = "Unreachable"
			} else {
				o.Status.Health = health.Status
			}
		}
	}

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o, data, logger)
}

func (h *ElasticsearchReconciler) Delete(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	// Read Cerebro referer to remove finalizer when destroy cluster
	hostList := &cerebrocrd.HostList{}
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef=%s", o.GetName()))
	if err = h.Client().List(ctx, hostList, &client.ListOptions{Namespace: o.GetNamespace(), FieldSelector: fs}); err != nil {
		return errors.Wrap(err, "error when read Cerebro hosts")
	}
	for _, host := range hostList.Items {
		controllerutil.RemoveFinalizer(&host, elasticsearchFinalizer.String())
		if err = h.Client().Update(ctx, &host); err != nil {
			return errors.Wrapf(err, "Error when delete finalizer on Host %s", host.Name)
		}
	}

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *ElasticsearchReconciler) OnError(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, currentErr error, logger *logrus.Entry) (res reconcile.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *ElasticsearchReconciler) OnSuccess(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	isReady := true

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

	// Check all statefulsets are ready to change Phase status and set main condition to true
	stsList := &appv1.StatefulSetList{}
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = h.Client().List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
		return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
	}

	if len(stsList.Items) == 0 {
		isReady = false
	}
loopStatefulset:
	for _, sts := range stsList.Items {
		for _, nodeGroup := range o.Spec.NodeGroups {
			if sts.Name == GetNodeGroupName(o, nodeGroup.Name) {
				if sts.Status.ReadyReplicas != nodeGroup.Replicas {
					isReady = false
					break loopStatefulset
				}
				break
			}
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

		if !o.IsBoostrapping() {
			o.Status.IsBootstrapping = ptr.To[bool](true)
		}

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

	o.Status.CredentialsRef = corev1.LocalObjectReference{
		Name: GetSecretNameForCredentials(o),
	}

	url, err := h.computeElasticsearchUrl(ctx, o)
	if err != nil {
		return res, err
	}
	o.Status.Url = url

	return res, nil
}

// computeElasticsearchUrl permit to get the public Elasticsearch url to put it on status
func (h *ElasticsearchReconciler) computeElasticsearchUrl(ctx context.Context, es *elasticsearchcrd.Elasticsearch) (target string, err error) {
	var (
		scheme string
		url    string
	)

	if es.IsIngressEnabled() {
		url = es.Spec.Endpoint.Ingress.Host

		if es.Spec.Endpoint.Ingress.SecretRef != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if es.IsRouteEnabled() {
		url = es.Spec.Endpoint.Route.Host

		if es.Spec.Endpoint.Route.TlsEnabled != nil && *es.Spec.Endpoint.Route.TlsEnabled {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else if es.IsLoadBalancerEnabled() {
		// Need to get lb service to get IP and port
		service := &corev1.Service{}
		if err = h.Client().Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: GetLoadBalancerName(es)}, service); err != nil {
			return "", errors.Wrap(err, "Error when get Load balancer")
		}

		if len(service.Status.LoadBalancer.Ingress) > 0 {
			url = fmt.Sprintf("%s:9200", service.Status.LoadBalancer.Ingress[0].IP)
		} else {
			return "", nil
		}

		if es.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else {
		url = fmt.Sprintf("%s.%s.svc:9200", GetGlobalServiceName(es), es.Namespace)
		if es.Spec.Tls.IsTlsEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return fmt.Sprintf("%s://%s", scheme, url), nil
}

func (h *ElasticsearchReconciler) getElasticsearchHandler(ctx context.Context, es *elasticsearchcrd.Elasticsearch, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {
	hosts := []string{}

	// Get Elasticsearch credentials
	secret := &corev1.Secret{}
	if err = h.Client().Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: GetSecretNameForCredentials(es)}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("Secret %s not yet exist, try later", GetSecretNameForCredentials(es))
			return nil, nil
		}
		log.Errorf("Error when get resource: %s", err.Error())
		return nil, err
	}

	serviceName := GetGlobalServiceName(es)
	if !es.Spec.Tls.IsTlsEnabled() {
		hosts = append(hosts, fmt.Sprintf("http://%s.%s.svc:9200", serviceName, es.Namespace))
	} else {
		hosts = append(hosts, fmt.Sprintf("https://%s.%s.svc:9200", serviceName, es.Namespace))
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ResponseHeaderTimeout: 10 * time.Second,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	cfg := elastic.Config{
		Transport: transport,
		Addresses: hosts,
		Username:  "elastic",
		Password:  string(secret.Data["elastic"]),
	}

	if log.Logger.GetLevel() == logrus.DebugLevel {
		cfg.Logger = &elastictransport.JSONLogger{EnableRequestBody: true, EnableResponseBody: true, Output: log.Logger.Out}
	}

	// Create Elasticsearch handler/client
	esHandler, err = eshandler.NewElasticsearchHandler(cfg, log)
	if err != nil {
		return nil, err
	}

	return esHandler, nil
}
