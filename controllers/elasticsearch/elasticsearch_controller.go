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
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elastic "github.com/elastic/go-elasticsearch/v8"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

const (
	name                   string               = "elasticsearch"
	elasticsearchFinalizer shared.FinalizerName = "elasticsearch.k8s.webcenter.fr/finalizer"
)

// ElasticsearchReconciler reconciles a Elasticsearch object
type ElasticsearchReconciler struct {
	controller.Controller
	controller.MultiPhaseReconcilerAction
	controller.MultiPhaseReconciler
	controller.BaseReconciler
	name string
}

// NewElasticsearchReconciler is the default constructor for Elasticsearch controller
func NewElasticsearchReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseReconciler controller.Controller) {

	multiPhaseReconciler = &ElasticsearchReconciler{
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
			elasticsearchFinalizer,
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Elasticsearch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ElasticsearchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	es := &elasticsearchcrd.Elasticsearch{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		es,
		data,
		r,
		newTlsReconciler(
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
		newStatefulsetReconciler(
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
		newSystemUserReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newLicenseReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newMetricbeatReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newExporterReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
		newPodMonitorReconciler(
			r.Client,
			r.Log,
			r.Recorder,
		),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *ElasticsearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
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
		Owns(&monitoringv1.PodMonitor{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(watchConfigMap(h.Client))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearchMonitoring(h.Client))).
		Watches(&cerebrocrd.Host{}, handler.EnqueueRequestsFromMapFunc(watchHost(h.Client))).
		Complete(h)
}

func (h *ElasticsearchReconciler) Configure(ctx context.Context, req ctrl.Request, resource object.MultiPhaseObject) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if o.Status.IsBootstrapping == nil {
		o.Status.IsBootstrapping = ptr.To[bool](false)
	}

	// Get Elasticsearch health
	// Not blocking way, cluster can be unreachable
	esHandler, err := h.getElasticsearchHandler(ctx, o, h.Log)
	if err != nil {
		h.Log.Warnf("Error when get elasticsearch client: %s", err.Error())
		o.Status.Health = "Unreachable"
	} else {
		if esHandler == nil {
			o.Status.Health = "Unreachable"
		} else {
			health, err := esHandler.ClusterHealth()
			if err != nil {
				h.Log.Warnf("Error when get elasticsearch health: %s", err.Error())
				o.Status.Health = "Unreachable"
			} else {
				o.Status.Health = health.Status
			}
		}
	}

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o)
}

func (h *ElasticsearchReconciler) Delete(ctx context.Context, o object.MultiPhaseObject, data map[string]any) (err error) {

	// Read Cerebro referer to remove finalizer when destroy cluster
	hostList := &cerebrocrd.HostList{}
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef=%s", o.GetName()))
	if err = h.Client.List(ctx, hostList, &client.ListOptions{Namespace: o.GetNamespace(), FieldSelector: fs}); err != nil {
		return errors.Wrap(err, "error when read Cerebro hosts")
	}
	for _, host := range hostList.Items {
		controllerutil.RemoveFinalizer(&host, elasticsearchFinalizer.String())
		if err = h.Client.Update(ctx, &host); err != nil {
			return errors.Wrapf(err, "Error when delete finalizer on Host %s", host.Name)
		}
	}

	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data)
}

func (h *ElasticsearchReconciler) OnError(ctx context.Context, o object.MultiPhaseObject, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr)
}

func (h *ElasticsearchReconciler) OnSuccess(ctx context.Context, r object.MultiPhaseObject, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*elasticsearchcrd.Elasticsearch)

	// Not preserve condition to avoid to update status each time
	conditions := o.GetStatus().GetConditions()
	o.GetStatus().SetConditions(nil)
	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data)
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
	if err = h.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
		return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
	}

	isReady := true
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
	} else if es.IsLoadBalancerEnabled() {
		// Need to get lb service to get IP and port
		service := &corev1.Service{}
		if err = h.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: GetLoadBalancerName(es)}, service); err != nil {
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
	if err = h.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: GetSecretNameForCredentials(es)}, secret); err != nil {
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
