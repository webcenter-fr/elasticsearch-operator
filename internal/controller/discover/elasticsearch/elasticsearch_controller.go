/*
Copyright 2024.

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

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	name                           string               = "elasticsearchDiscover"
	elasticsearchDiscoverFinalizer shared.FinalizerName = "elasticsearch.discover.k8s.webcenter.fr/finalizer"
)

// ElasticsearchDiscoverReconciler reconciles an Elasticsearch discover object
type ElasticsearchDiscoverReconciler struct {
	controller.Controller
	multiphase.MultiPhaseReconciler[*discovercrd.Elasticsearch]
	multiphase.MultiPhaseReconcilerAction[*discovercrd.Elasticsearch]
	name            string
	stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Elasticsearch, client.Object]
	kubeCapability  common.KubernetesCapability
}

func NewElasticsearchDiscoverReconciler(c client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &ElasticsearchDiscoverReconciler{
		Controller: controller.NewController(),
		MultiPhaseReconciler: multiphase.NewMultiPhaseReconciler[*discovercrd.Elasticsearch](
			c,
			name,
			elasticsearchDiscoverFinalizer,
			logger,
			recorder,
		),
		MultiPhaseReconcilerAction: multiphase.NewMultiPhaseReconcilerAction[*discovercrd.Elasticsearch](
			c,
			controller.ReadyCondition,
			recorder,
		),

		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Elasticsearch, client.Object]{
			multiphase.As[*discovercrd.Elasticsearch, *corev1.Secret, client.Object](newSecretElasticsearchReconciler(c, recorder)),
		},
	}

	return reconciler
}

//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=elasticsearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=elasticsearches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=elasticsearches/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ElasticsearchDiscoverReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	o := &discovercrd.Elasticsearch{}
	data := map[string]any{}

	return r.MultiPhaseReconciler.Reconcile(
		ctx,
		req,
		o,
		data,
		r,
		r.stepReconcilers...,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (h *ElasticsearchDiscoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		Named("discoverElasticsearch").
		For(&discovercrd.Elasticsearch{}).
		Owns(&corev1.Secret{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		Watches(&elasticsearchcrd.Elasticsearch{}, handler.EnqueueRequestsFromMapFunc(watchElasticsearch(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		})

	return ctrlBuilder.Complete(h)
}

func (h *ElasticsearchDiscoverReconciler) Client() client.Client {
	return h.MultiPhaseReconcilerAction.Client()
}

func (h *ElasticsearchDiscoverReconciler) Recorder() record.EventRecorder {
	return h.MultiPhaseReconcilerAction.Recorder()
}

func (h *ElasticsearchDiscoverReconciler) Configure(ctx context.Context, req reconcile.Request, o *discovercrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(1)

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o, data, logger)
}

func (h *ElasticsearchDiscoverReconciler) Delete(ctx context.Context, o *discovercrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *ElasticsearchDiscoverReconciler) OnError(ctx context.Context, o *discovercrd.Elasticsearch, data map[string]any, currentErr error, logger *logrus.Entry) (res reconcile.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *ElasticsearchDiscoverReconciler) OnSuccess(ctx context.Context, o *discovercrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Reset the current cluster errors
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data, logger)
	if err != nil {
		return res, err
	}

	o.Status.SecretEnvRef = ptr.To(GetSecretNameForEnv(o))
	o.Status.SecretFileRef = ptr.To(GetSecretNameForFile(o))
	if data["elasticsearchCASecretRef"] != nil {
		o.Status.ElasticsearchCASecretRef = data["elasticsearchCASecretRef"].(*string)
	}
	if data["elasticsearchUserSecretRef"] != nil {
		o.Status.ElasticsearchUserSecretRef = data["elasticsearchUserSecretRef"].(*string)
	}

	return res, nil
}
