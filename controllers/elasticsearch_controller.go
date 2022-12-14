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

package controllers

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	elasticsearchv1alpha1 "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	esctrl "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ElasticsearchFinalizer = "elasticsearch.k8s.webcenter.fr/finalizer"
	ElasticsearchCondition = "ElasticsearchReady"
	ElasticsearchPhase     = "running"
)

// ElasticsearchReconciler reconciles a Elasticsearch object
type ElasticsearchReconciler struct {
	Reconciler
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

	controllerMetrics.WithLabelValues(r.name).Add(0)

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
	reconciler, err := controller.NewStdK8sReconciler(r.Client, ElasticsearchFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	es := &elasticsearchv1alpha1.Elasticsearch{}
	data := map[string]any{}

	tlsReconsiler := esctrl.NewTlsReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "tls",
		}),
	})

	configmapReconciler := esctrl.NewConfiMapReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "configmap",
		}),
	})

	serviceReconciler := esctrl.NewServiceReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "service",
		}),
	})

	ingressReconciler := esctrl.NewIngressReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "ingress",
		}),
	})

	loadBalancerReconciler := esctrl.NewLoadBalancerReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "loadBalancer",
		}),
	})

	pdbReconciler := esctrl.NewPdbReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "podDisruptionBudget",
		}),
	})

	credentialReconciler := esctrl.NewCredentialReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "credential",
		}),
	})

	statefulsetReconciler := esctrl.NewStatefulsetReconciler(r.Client, r.Scheme, common.Reconciler{
		Recorder: r.recorder,
		Log: r.log.WithFields(logrus.Fields{
			"phase": "statefulset",
		}),
	})

	return reconciler.Reconcile(ctx, req, es, data,
		tlsReconsiler,
		credentialReconciler,
		configmapReconciler,
		serviceReconciler,
		pdbReconciler,
		statefulsetReconciler,
		ingressReconciler,
		loadBalancerReconciler,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *ElasticsearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchv1alpha1.Elasticsearch{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&appv1.StatefulSet{}).
		Complete(h)
}

func (h *ElasticsearchReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ElasticsearchCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ElasticsearchCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	return res, nil
}
func (h *ElasticsearchReconciler) Read(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	return
}

func (h *ElasticsearchReconciler) Delete(ctx context.Context, r client.Object, data map[string]any) (err error) {
	controllerMetrics.WithLabelValues(h.name).Dec()
	return
}
func (h *ElasticsearchReconciler) OnError(ctx context.Context, r client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	totalErrors.Inc()
	return res, currentErr
}
func (h *ElasticsearchReconciler) OnSuccess(ctx context.Context, r client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := r.(*elasticsearchapi.Elasticsearch)

	// Wait few time, to be sure Satefulset created
	time.Sleep(1 * time.Second)

	// Check all statefulsets are ready to change Phase status and set main condition to true
	stsList := &appv1.StatefulSetList{}
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearch.ElasticsearchAnnotationKey))
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

			o.Status.Phase = "running"
		}

		return res, nil
	}

	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ElasticsearchCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ElasticsearchCondition,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})

	}

	o.Status.Phase = "starting"

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}
func (h *ElasticsearchReconciler) Diff(ctx context.Context, r client.Object, data map[string]any) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return
}
func (h *ElasticsearchReconciler) Name() string {
	return "elasticsearch"
}
