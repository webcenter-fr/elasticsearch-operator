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
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	snapshotRepositoryName string = "snapshotRepository"
)

// SnapshotRepositoryReconciler reconciles a SnapshotRepository object
type SnapshotRepositoryReconciler struct {
	controller.Controller
	controller.RemoteReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler]
	reconcilerAction controller.RemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler]
	name             string
}

func NewSnapshotRepositoryReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.Controller {
	r := &SnapshotRepositoryReconciler{
		Controller: controller.NewBasicController(),
		RemoteReconciler: controller.NewBasicRemoteReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](
			client,
			snapshotRepositoryName,
			"snapshotrepository.elasticsearchapi.k8s.webcenter.fr/finalizer",
			logger,
			recorder,
		),
		reconcilerAction: newSnapshotRepositoryReconciler(
			client,
			logger,
			recorder,
		),
		name: snapshotRepositoryName,
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotrepositories/finalizers,verbs=update
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
func (r *SnapshotRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sr := &elasticsearchapicrd.SnapshotRepository{}
	data := map[string]any{}

	return r.RemoteReconciler.Reconcile(
		ctx,
		req,
		sr,
		data,
		r.reconcilerAction,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.SnapshotRepository{}).
		Complete(r)
}
