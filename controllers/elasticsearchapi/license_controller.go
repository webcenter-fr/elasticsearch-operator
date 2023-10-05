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

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	licenseName string = "license"
)

// LicenseReconciler reconciles a License object
type LicenseReconciler struct {
	controller.BaseReconciler
	controller.Controller
	controller.RemoteReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense]
	reconcilerAction controller.RemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense]
	name             string
}

func NewLicenseReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {

	r := &LicenseReconciler{
		Controller: controller.NewBasicController(),
		RemoteReconciler: controller.NewBasicRemoteReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense](
			client,
			licenseName,
			"license.elasticsearchapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		reconcilerAction: newLicenseReconciler(
			client,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Log:      logger,
			Recorder: recorder,
		},
		name: licenseName,
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
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
func (r *LicenseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	license := &elasticsearchapicrd.License{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		license,
		data,
		r.reconcilerAction,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LicenseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.License{}).
		Watches(&core.Secret{}, handler.EnqueueRequestsFromMapFunc(watchLicenseSecret(r.Client))).
		Complete(r)
}

// watchLicenseSecret permit to update license if secret change
func watchLicenseSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {

		reconcileRequests := make([]reconcile.Request, 0)
		listLicenses := &elasticsearchapicrd.LicenseList{}

		fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.secretRef.name=%s", a.GetName()))

		// Get all license linked with secret
		if err := c.List(context.Background(), listLicenses, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}

		for _, l := range listLicenses.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: l.Name, Namespace: l.Namespace}})
		}

		return reconcileRequests
	}
}
