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

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	indexLifecyclePolicyName string = "indexLifecyclePolicy"
)

// IndexLifecyclePolicyReconciler reconciles a IndexLifecyclePolicy object
type IndexLifecyclePolicyReconciler struct {
	controller.Controller
	controller.RemoteReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse]
	reconcilerAction controller.RemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse]
	name             string
}

func NewIndexLifecyclePolicyReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {

	r := &IndexLifecyclePolicyReconciler{
		Controller: controller.NewBasicController(),
		RemoteReconciler: controller.NewBasicRemoteReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse](
			client,
			indexLifecyclePolicyName,
			"ilm.elasticsearchapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		reconcilerAction: newIndexLifecyclePolicyReconciler(
			client,
			logger,
			recorder,
		),
		name: indexLifecyclePolicyName,
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indexlifecyclepolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indexlifecyclepolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indexlifecyclepolicies/finalizers,verbs=update
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
func (r *IndexLifecyclePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ilm := &elasticsearchapicrd.IndexLifecyclePolicy{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		ilm,
		data,
		r.reconcilerAction,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexLifecyclePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.IndexLifecyclePolicy{}).
		Complete(r)
}
