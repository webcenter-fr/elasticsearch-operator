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
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elastic "github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ElasticsearchFinalizer     = "elasticsearch.k8s.webcenter.fr/finalizer"
	ElasticsearchCondition     = "ElasticsearchReady"
	ElasticsearchPhaseRunning  = "running"
	ElasticsearchPhaseStarting = "starting"
)

// ElasticsearchReconciler reconciles a Elasticsearch object
type ElasticsearchReconciler struct {
	common.Controller
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewElasticsearchReconciler(client client.Client, scheme *runtime.Scheme) *ElasticsearchReconciler {

	r := &ElasticsearchReconciler{
		Client: client,
		Scheme: scheme,
		name:   "elasticsearch",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearch.k8s.webcenter.fr,resources=elasticsearches/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/finalizers,verbs=update

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
	reconciler, err := controller.NewStdK8sReconciler(r.Client, ElasticsearchFinalizer, r.GetReconciler(), r.GetLogger(), r.GetRecorder())
	if err != nil {
		return ctrl.Result{}, err
	}

	es := &elasticsearchcrd.Elasticsearch{}
	data := map[string]any{}

	tlsReconsiler := NewTlsReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "tls",
		}),
	})

	configmapReconciler := NewConfiMapReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "configmap",
		}),
	})

	serviceReconciler := NewServiceReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "service",
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

	pdbReconciler := NewPdbReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "podDisruptionBudget",
		}),
	})

	networkPolicyReconciler := NewNetworkPolicyReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "networkPolicy",
		}),
	})

	credentialReconciler := NewCredentialReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "credential",
		}),
	})

	statefulsetReconciler := NewStatefulsetReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "statefulset",
		}),
	})

	userReconciler := NewSystemUserReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "systemUser",
		}),
	})

	licenseReconciler := NewLicenseReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.GetRecorder(),
		Log: r.GetLogger().WithFields(logrus.Fields{
			"phase": "license",
		}),
	})

	return reconciler.Reconcile(ctx, req, es, data,
		tlsReconsiler,
		credentialReconciler,
		configmapReconciler,
		serviceReconciler,
		pdbReconciler,
		networkPolicyReconciler,
		statefulsetReconciler,
		ingressReconciler,
		loadBalancerReconciler,
		userReconciler,
		licenseReconciler,
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
		Owns(&elasticsearchapicrd.User{}).
		Owns(&elasticsearchapicrd.License{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client))).
		Complete(h)
}

// watchSecret permit to update elasticsearch if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		var (
			listElasticsearch *elasticsearchcrd.ElasticsearchList
			fs                fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Keystore secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.globalNodeGroup.keystoreSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// TLS secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.tls.certificateSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		return reconcileRequests
	}
}

func (h *ElasticsearchReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ElasticsearchCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ElasticsearchCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get Elasticsearch health
	// Not blocking way, cluster can be unreachable
	esHandler, err := h.getElasticsearchHandler(ctx, o, h.GetLogger())
	if err != nil {
		h.GetLogger().Warnf("Error when get elasticsearch client: %s", err.Error())
		o.Status.Health = "Unreachable"

		return res, nil
	}

	if esHandler == nil {
		o.Status.Health = "Unreachable"
	} else {
		health, err := esHandler.ClusterHealth()
		if err != nil {
			h.GetLogger().Warnf("Error when get elasticsearch health: %s", err.Error())
			o.Status.Health = "Unreachable"
			return res, nil
		}

		if o.Status.Health != health.Status {
			o.Status.Health = health.Status
		}
	}

	return res, nil
}
func (h *ElasticsearchReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *ElasticsearchReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	common.ControllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *ElasticsearchReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	common.TotalErrors.Inc()
	return res, currentErr
}
func (h *ElasticsearchReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*elasticsearchcrd.Elasticsearch)

	// Wait few time, to be sure Satefulset created
	time.Sleep(1 * time.Second)

	// Check all statefulsets are ready to change Phase status and set main condition to true
	stsList := &appv1.StatefulSetList{}
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = h.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
		return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
	}

	isReady := true
	for _, sts := range stsList.Items {
		if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
			isReady = false
			break
		}
	}

	if isReady {
		if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ElasticsearchCondition, metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:   ElasticsearchCondition,
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			})
		}

		if o.Status.Phase != ElasticsearchPhaseRunning {
			o.Status.Phase = ElasticsearchPhaseRunning
		}

	} else if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ElasticsearchCondition, metav1.ConditionFalse) || (condition.FindStatusCondition(o.Status.Conditions, ElasticsearchCondition) != nil && condition.FindStatusCondition(o.Status.Conditions, ElasticsearchCondition).Reason != "NotReady") {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ElasticsearchCondition,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})

		if o.Status.Phase != ElasticsearchPhaseStarting {
			o.Status.Phase = ElasticsearchPhaseStarting
		}
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

func (h *ElasticsearchReconciler) Name() string {
	return "elasticsearch"
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

		url = fmt.Sprintf("%s:9200", service.Spec.LoadBalancerIP)
		if es.IsTlsApiEnabled() {
			scheme = "https"
		} else {
			scheme = "http"
		}
	} else {
		url = fmt.Sprintf("%s.%s.svc:9200", GetGlobalServiceName(es), es.Namespace)
		if es.IsTlsApiEnabled() {
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
	if !es.IsTlsApiEnabled() {
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
