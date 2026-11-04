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

package logstash

import (
	"context"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
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
	name                      string               = "logstashDiscover"
	logstashDiscoverFinalizer shared.FinalizerName = "logstash.discover.k8s.webcenter.fr/finalizer"
)

// LogstashDiscoverReconciler reconciles a Logstash object
type LogstashDiscoverReconciler struct {
	controller.Controller
	multiphase.MultiPhaseReconciler[*discovercrd.Logstash]
	multiphase.MultiPhaseReconcilerAction[*discovercrd.Logstash]
	name            string
	stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Logstash, client.Object]
	kubeCapability  common.KubernetesCapability
}

func NewLogstashDiscoverReconciler(c client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &LogstashDiscoverReconciler{
		Controller: controller.NewController(),
		MultiPhaseReconciler: multiphase.NewMultiPhaseReconciler[*discovercrd.Logstash](
			c,
			name,
			logstashDiscoverFinalizer,
			logger,
			recorder,
		),
		MultiPhaseReconcilerAction: multiphase.NewMultiPhaseReconcilerAction[*discovercrd.Logstash](
			c,
			controller.ReadyCondition,
			recorder,
		),

		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Logstash, client.Object]{
			multiphase.As[*discovercrd.Logstash, *corev1.Secret, client.Object](newSecretLogstashReconciler(c, recorder)),
		},
	}

	return reconciler
}

//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=logstashes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=logstashes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=logstashes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="logstash.k8s.webcenter.fr",resources=logstashes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LogstashDiscoverReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	o := &discovercrd.Logstash{}
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
func (h *LogstashDiscoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		Named("discoverLogstash").
		For(&discovercrd.Logstash{}).
		Owns(&corev1.Secret{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		Watches(&logstashcrd.Logstash{}, handler.EnqueueRequestsFromMapFunc(watchLogstash(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		})

	return ctrlBuilder.Complete(h)
}

func (h *LogstashDiscoverReconciler) Client() client.Client {
	return h.MultiPhaseReconcilerAction.Client()
}

func (h *LogstashDiscoverReconciler) Recorder() record.EventRecorder {
	return h.MultiPhaseReconcilerAction.Recorder()
}

func (h *LogstashDiscoverReconciler) Configure(ctx context.Context, req reconcile.Request, o *discovercrd.Logstash, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(1)

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o, data, logger)
}

func (h *LogstashDiscoverReconciler) Delete(ctx context.Context, o *discovercrd.Logstash, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *LogstashDiscoverReconciler) OnError(ctx context.Context, o *discovercrd.Logstash, data map[string]any, currentErr error, logger *logrus.Entry) (res reconcile.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *LogstashDiscoverReconciler) OnSuccess(ctx context.Context, o *discovercrd.Logstash, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Reset the current cluster errors
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data, logger)
	if err != nil {
		return res, err
	}

	o.Status.SecretEnvRef = ptr.To(GetSecretNameForEnv(o))
	o.Status.SecretFileRef = ptr.To(GetSecretNameForFile(o))
	if data["logstashCASecretRef"] != nil {
		o.Status.LogstashCASecretRef = data["logstashCASecretRef"].(*string)
	}
	if data["logstashUserSecretRef"] != nil {
		o.Status.LogstashUserSecretRef = data["logstashUserSecretRef"].(*string)
	}

	return res, nil
}
