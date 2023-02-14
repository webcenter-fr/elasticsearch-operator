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
	"encoding/json"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	olivere "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	core "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SnapshotRepositoryFinalizer = "snapshotrepository.elasticsearchapi.k8s.webcenter.fr/finalizer"
	SnapshotRepositoryCondition = "SnapshotRepository"
)

// SnapshotRepositoryReconciler reconciles a SnapshotRepository object
type SnapshotRepositoryReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewSnapshotRepositoryReconciler(client client.Client, scheme *runtime.Scheme) *SnapshotRepositoryReconciler {

	r := &SnapshotRepositoryReconciler{
		Client: client,
		Scheme: scheme,
		name:   "snapshotRepository",
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

	reconciler, err := controller.NewStdReconciler(r.Client, SnapshotRepositoryFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	sr := &elasticsearchapicrd.SnapshotRepository{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, sr, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.SnapshotRepository{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *SnapshotRepositoryReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	// Init condition status if not exist
	if condition.FindStatusCondition(sr.Status.Conditions, SnapshotRepositoryCondition) == nil {
		condition.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
			Type:   SnapshotRepositoryCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, sr.Spec.ElasticsearchRef, r.Client, req, r.log)
	if err != nil {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, err
}

// Read permit to get current snapshot repository
func (r *SnapshotRepositoryReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if sr.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read snapshot repository from Elasticsearch
	currentRepository, err := esHandler.SnapshotRepositoryGet(sr.GetSnapshotRepositoryName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get snapshot repository from Elasticsearch")
	}
	data["current"] = currentRepository

	// Generate expected
	settings := map[string]any{}
	if err = json.Unmarshal([]byte(sr.Spec.Settings), &settings); err != nil {
		return res, errors.Wrap(err, "Unable to generate snapshot repository")
	}
	expectedRepository := &olivere.SnapshotRepositoryMetaData{
		Type:     sr.Spec.Type,
		Settings: settings,
	}
	data["expected"] = expectedRepository

	return res, nil
}

// Create add new snapshot repository
func (r *SnapshotRepositoryReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	settings := map[string]any{}
	if err = json.Unmarshal([]byte(sr.Spec.Settings), &settings); err != nil {
		return res, errors.Wrap(err, "Error when decode repository setting")
	}

	repoObj := &olivere.SnapshotRepositoryMetaData{
		Type:     sr.Spec.Type,
		Settings: settings,
	}

	// Create repository on Elasticsearch
	if err = esHandler.SnapshotRepositoryUpdate(sr.GetSnapshotRepositoryName(), repoObj); err != nil {
		return res, errors.Wrap(err, "Error when update snapshot repository")
	}

	return res, nil
}

// Update permit to update current snapshot repository from Elasticsearch
func (r *SnapshotRepositoryReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete snapshot repository from Elasticsearch
func (r *SnapshotRepositoryReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	if err = esHandler.SnapshotRepositoryDelete(sr.GetSnapshotRepositoryName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch snapshot repository")
	}

	return nil

}

// Diff permit to check if diff between actual and expected snapshot repository exist
func (r *SnapshotRepositoryReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentSR := d.(*olivere.SnapshotRepositoryMetaData)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedSR := d.(*olivere.SnapshotRepositoryMetaData)

	var originalSR *olivere.SnapshotRepositoryMetaData
	if sr.Status.OriginalObject != "" {
		originalSR = &olivere.SnapshotRepositoryMetaData{}
		if err = localhelper.UnZipBase64Decode(sr.Status.OriginalObject, originalSR); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentSR == nil {
		diff.NeedCreate = true
		diff.Diff = "Snapshot repository not exist"

		if err = localhelper.SetLastOriginal(sr, expectedSR); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.SnapshotRepositoryDiff(currentSR, expectedSR, originalSR)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(sr, expectedSR); err != nil {
			return diff, err
		}

		return diff, nil
	}
	return
}

// OnError permit to set status condition on the right state and record error
func (r *SnapshotRepositoryReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
		Type:    SnapshotRepositoryCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	sr.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *SnapshotRepositoryReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	sr := resource.(*elasticsearchapicrd.SnapshotRepository)

	sr.Status.Sync = true

	if diff.NeedCreate {
		condition.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
			Type:    SnapshotRepositoryCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Snapshot repository successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Snapshot repository successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
			Type:    SnapshotRepositoryCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Snapshot repository successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Snapshot repository successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(sr.Status.Conditions, SnapshotRepositoryCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&sr.Status.Conditions, metav1.Condition{
			Type:    SnapshotRepositoryCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Snapshot repository already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Snapshot repository already set")
	}

	return nil
}
