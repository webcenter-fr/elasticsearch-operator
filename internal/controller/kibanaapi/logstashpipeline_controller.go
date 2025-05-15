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

package kibanaapi

import (
	"context"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/sirupsen/logrus"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	logstashPipelineName string = "logstashPipeline"
)

// LogstashPipelineReconciler reconciles a Logstash pipeline object
type LogstashPipelineReconciler struct {
	controller.Controller
	remote.RemoteReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler]
	remote.RemoteReconcilerAction[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler]
	name string
}

func NewLogstashPipelineReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {
	return &LogstashPipelineReconciler{
		Controller: controller.NewController(),
		RemoteReconciler: remote.NewRemoteReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler](
			client,
			logstashPipelineName,
			"pipeline.kibanaapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		RemoteReconcilerAction: newLogstashPipelineReconciler(
			logstashPipelineName,
			client,
			recorder,
		),
		name: logstashPipelineName,
	}
}

//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get
//+kubebuilder:rbac:groups="kibana.k8s.webcenter.fr",resources=kibanas,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Logstash Pipeline object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LogstashPipelineReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	pipeline := &kibanaapicrd.LogstashPipeline{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		pipeline,
		data,
		r,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LogstashPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanaapicrd.LogstashPipeline{}).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		}).
		Complete(r)
}

func (h *LogstashPipelineReconciler) Client() client.Client {
	return h.RemoteReconcilerAction.Client()
}

func (h *LogstashPipelineReconciler) Recorder() record.EventRecorder {
	return h.RemoteReconcilerAction.Recorder()
}
