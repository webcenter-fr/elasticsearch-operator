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
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	olivere "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
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
	WatchFinalizer = "watch.elasticsearchapi.k8s.webcenter.fr/finalizer"
	WatchCondition = "Watch"
)

// WatchReconciler reconciles a Watch object
type WatchReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewWatchReconciler(client client.Client, scheme *runtime.Scheme) *WatchReconciler {

	r := &WatchReconciler{
		Client: client,
		Scheme: scheme,
		name:   "watch",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
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
func (r *WatchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, WatchFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	watch := &elasticsearchapicrd.Watch{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, watch, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.Watch{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *WatchReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	watch := resource.(*elasticsearchapicrd.Watch)

	// Init condition status if not exist
	if condition.FindStatusCondition(watch.Status.Conditions, WatchCondition) == nil {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:   WatchCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(watch.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, watch, watch.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil && watch.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current watch
func (r *WatchReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	watch := resource.(*elasticsearchapicrd.Watch)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if watch.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read watch from Elasticsearch
	currentWatch, err := esHandler.WatchGet(watch.GetWatchName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get watch from Elasticsearch")
	}
	data["current"] = currentWatch

	// Generate expected
	expectedWatch, err := BuildWatch(watch)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate watch")
	}
	data["expected"] = expectedWatch

	return res, nil
}

// Create add new watch
func (r *WatchReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	watch := resource.(*elasticsearchapicrd.Watch)

	// Create watch on Elasticsearch
	expectedWatch, err := BuildWatch(watch)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to watch")
	}
	if err = esHandler.WatchUpdate(watch.GetWatchName(), expectedWatch); err != nil {
		return res, errors.Wrap(err, "Error when update watch")
	}
	return res, nil
}

// Update permit to update current watch from Elasticsearch
func (r *WatchReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete watch from Elasticsearch
func (r *WatchReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	watch := resource.(*elasticsearchapicrd.Watch)

	if err = esHandler.WatchDelete(watch.GetWatchName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch watch")
	}

	return nil

}

// Diff permit to check if diff between actual and expected watch exist
func (r *WatchReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	watch := resource.(*elasticsearchapicrd.Watch)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentWatch := d.(*olivere.XPackWatch)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedWatch := d.(*olivere.XPackWatch)

	var originalWatch *olivere.XPackWatch
	if watch.Status.OriginalObject != "" {
		originalWatch = &olivere.XPackWatch{}
		if err = localhelper.UnZipBase64Decode(watch.Status.OriginalObject, originalWatch); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentWatch == nil {
		diff.NeedCreate = true
		diff.Diff = "Watch not exist"

		if err = localhelper.SetLastOriginal(watch, expectedWatch); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.WatchDiff(currentWatch, expectedWatch, originalWatch)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(watch, expectedWatch); err != nil {
			return diff, err
		}

		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *WatchReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	watch := resource.(*elasticsearchapicrd.Watch)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
		Type:    WatchCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	watch.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *WatchReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	watch := resource.(*elasticsearchapicrd.Watch)

	watch.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(watch.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:    WatchCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Watch successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Watch successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:    WatchCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Watch successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Watch successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(watch.Status.Conditions, WatchCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&watch.Status.Conditions, metav1.Condition{
			Type:    WatchCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Watch already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Watch already set")
	}

	return nil
}
