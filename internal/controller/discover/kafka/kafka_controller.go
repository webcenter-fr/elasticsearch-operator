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

package kafka

import (
	"context"

	strimzicrd "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
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
	name                   string               = "kafkaDiscover"
	kafkaDiscoverFinalizer shared.FinalizerName = "kafka.discover.k8s.webcenter.fr/finalizer"
)

// KafkaDiscoverReconciler reconciles a Kafka object
type KafkaDiscoverReconciler struct {
	controller.Controller
	multiphase.MultiPhaseReconciler[*discovercrd.Kafka]
	multiphase.MultiPhaseReconcilerAction[*discovercrd.Kafka]
	name            string
	stepReconcilers []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Kafka, client.Object]
	kubeCapability  common.KubernetesCapability
}

func NewKafkaDiscoverReconciler(c client.Client, logger *logrus.Entry, recorder record.EventRecorder, kubeCapability common.KubernetesCapability) (multiPhaseReconciler controller.Controller) {
	reconciler := &KafkaDiscoverReconciler{
		Controller: controller.NewController(),
		MultiPhaseReconciler: multiphase.NewMultiPhaseReconciler[*discovercrd.Kafka](
			c,
			name,
			kafkaDiscoverFinalizer,
			logger,
			recorder,
		),
		MultiPhaseReconcilerAction: multiphase.NewMultiPhaseReconcilerAction[*discovercrd.Kafka](
			c,
			controller.ReadyCondition,
			recorder,
		),

		name:           name,
		kubeCapability: kubeCapability,
		stepReconcilers: []multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Kafka, client.Object]{
			multiphase.As[*discovercrd.Kafka, *corev1.Secret, client.Object](newSecretKafkaReconciler(c, recorder, kubeCapability.HasStrimzi)),
		},
	}

	return reconciler
}

//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=kafkas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=discover.k8s.webcenter.fr,resources=kafkas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="kafka.strimzi.io",resources=kafkas,verbs=get;list;watch
//+kubebuilder:rbac:groups="kafka.strimzi.io",resources=kafkausers,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KafkaDiscoverReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	o := &discovercrd.Kafka{}
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
func (h *KafkaDiscoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		Named("discoverKafka").
		For(&discovercrd.Kafka{}).
		Owns(&corev1.Secret{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(watchSecret(h.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		})

	if h.kubeCapability.HasStrimzi {
		ctrlBuilder.Watches(&strimzicrd.Kafka{}, handler.EnqueueRequestsFromMapFunc(watchKafka(h.Client())))
		ctrlBuilder.Watches(&strimzicrd.KafkaUser{}, handler.EnqueueRequestsFromMapFunc(watchKafkaUser(h.Client())))
	}

	return ctrlBuilder.Complete(h)
}

func (h *KafkaDiscoverReconciler) Client() client.Client {
	return h.MultiPhaseReconcilerAction.Client()
}

func (h *KafkaDiscoverReconciler) Recorder() record.EventRecorder {
	return h.MultiPhaseReconcilerAction.Recorder()
}

func (h *KafkaDiscoverReconciler) Configure(ctx context.Context, req reconcile.Request, o *discovercrd.Kafka, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(1)

	return h.MultiPhaseReconcilerAction.Configure(ctx, req, o, data, logger)
}

func (h *KafkaDiscoverReconciler) Delete(ctx context.Context, o *discovercrd.Kafka, data map[string]any, logger *logrus.Entry) (err error) {
	// Set prometheus Metrics
	common.ControllerInstances.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	return h.MultiPhaseReconcilerAction.Delete(ctx, o, data, logger)
}

func (h *KafkaDiscoverReconciler) OnError(ctx context.Context, o *discovercrd.Kafka, data map[string]any, currentErr error, logger *logrus.Entry) (res reconcile.Result, err error) {
	common.TotalErrors.Inc()
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Inc()

	return h.MultiPhaseReconcilerAction.OnError(ctx, o, data, currentErr, logger)
}

func (h *KafkaDiscoverReconciler) OnSuccess(ctx context.Context, o *discovercrd.Kafka, data map[string]any, logger *logrus.Entry) (res reconcile.Result, err error) {
	// Reset the current cluster errors
	common.ControllerErrors.WithLabelValues(h.name, o.GetNamespace(), o.GetName()).Set(0)

	res, err = h.MultiPhaseReconcilerAction.OnSuccess(ctx, o, data, logger)
	if err != nil {
		return res, err
	}

	o.Status.SecretEnvRef = ptr.To(GetSecretNameForEnv(o))
	o.Status.SecretFileRef = ptr.To(GetSecretNameForFile(o))
	if data["kafkaUserSecretRef"] != nil {
		o.Status.KafkaUserSecretRef = data["kafkaUserSecretRef"].(*string)
	}
	if data["kafkaCASecretRef"] != nil {
		o.Status.KafkaCASecretRef = data["kafkaCASecretRef"].(*string)
	}

	return res, nil
}
