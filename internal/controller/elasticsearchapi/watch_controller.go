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

package elasticsearchapi

import (
	"context"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	watchName string = "watch"
)

// WatchReconciler reconciles a Watch object
type WatchReconciler struct {
	controller.Controller
	remote.RemoteReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler]
	remote.RemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler]
	name string
}

func NewWatchReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {
	return &WatchReconciler{
		Controller: controller.NewController(),
		RemoteReconciler: remote.NewRemoteReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler](
			client,
			watchName,
			"watch.elasticsearchapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		RemoteReconcilerAction: newWatchReconciler(
			watchName,
			client,
			recorder,
		),
		name: watchName,
	}
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=watches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=watches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=watches/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the License object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *WatchReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	watch := &elasticsearchapicrd.Watch{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		watch,
		data,
		r,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.Watch{}).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		}).
		Complete(r)
}

func (h *WatchReconciler) Client() client.Client {
	return h.RemoteReconcilerAction.Client()
}

func (h *WatchReconciler) Recorder() record.EventRecorder {
	return h.RemoteReconcilerAction.Recorder()
}
