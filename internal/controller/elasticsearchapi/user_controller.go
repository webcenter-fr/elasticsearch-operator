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
	"fmt"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	userName string = "user"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	controller.Controller
	remote.RemoteReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]
	remote.RemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]
	name string
}

func NewUserReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {
	return &UserReconciler{
		Controller: controller.NewController(),
		RemoteReconciler: remote.NewRemoteReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler](
			client,
			userName,
			"user.elasticsearchapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		RemoteReconcilerAction: newUserReconciler(
			userName,
			client,
			recorder,
		),
		name: userName,
	}
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	user := &elasticsearchapicrd.User{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		user,
		data,
		r,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.User{}).
		Watches(&core.Secret{}, handler.EnqueueRequestsFromMapFunc(watchUserSecret(r.Client()))).
		WithOptions(k8scontroller.Options{
			RateLimiter: controller.DefaultControllerRateLimiter[reconcile.Request](),
		}).
		Complete(r)
}

func (h *UserReconciler) Client() client.Client {
	return h.RemoteReconcilerAction.Client()
}

func (h *UserReconciler) Recorder() record.EventRecorder {
	return h.RemoteReconcilerAction.Recorder()
}

// watchUserSecret permit to update user if secret change
func watchUserSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		reconcileRequests := make([]reconcile.Request, 0)
		listUsers := &elasticsearchapicrd.UserList{}

		fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.secretRef.name=%s", a.GetName()))

		// Get all users linked with secret
		if err := c.List(context.Background(), listUsers, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}

		for _, u := range listUsers.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: u.Name, Namespace: u.Namespace}})
		}

		return reconcileRequests
	}
}
