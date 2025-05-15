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
	userSpaceName string = "UserSpace"
)

// UserSpaceReconciler reconciles a user space object
type UserSpaceReconciler struct {
	controller.Controller
	remote.RemoteReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler]
	remote.RemoteReconcilerAction[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler]
	name string
}

func NewUserSpaceReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {
	return &UserSpaceReconciler{
		Controller: controller.NewController(),
		RemoteReconciler: remote.NewRemoteReconciler[*kibanaapicrd.UserSpace, *kbapi.KibanaSpace, kbhandler.KibanaHandler](
			client,
			userSpaceName,
			"space.kibanaapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		RemoteReconcilerAction: newUserSpaceReconciler(
			userSpaceName,
			client,
			recorder,
		),
		name: userSpaceName,
	}
}

//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get
//+kubebuilder:rbac:groups="kibana.k8s.webcenter.fr",resources=kibanas,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UserSpace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *UserSpaceReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	space := &kibanaapicrd.UserSpace{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		space,
		data,
		r,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserSpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanaapicrd.UserSpace{}).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		}).
		Complete(r)
}

func (h *UserSpaceReconciler) Client() client.Client {
	return h.RemoteReconcilerAction.Client()
}

func (h *UserSpaceReconciler) Recorder() record.EventRecorder {
	return h.RemoteReconcilerAction.Recorder()
}
