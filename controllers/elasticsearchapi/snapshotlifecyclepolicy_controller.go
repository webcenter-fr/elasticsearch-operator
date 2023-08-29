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

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
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
	SnapshotLifecyclePolicyFinalizer = "slm.elasticsearchapi.k8s.webcenter.fr/finalizer"
	SnapshotLifecyclePolicyCondition = "SnapshotLifecyclePolicy"
)

// SnapshotLifecyclePolicyReconciler reconciles a SnapshotLifecyclePolicy object
type SnapshotLifecyclePolicyReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewSnapshotLifecyclePolicyReconciler(client client.Client, scheme *runtime.Scheme) *SnapshotLifecyclePolicyReconciler {

	r := &SnapshotLifecyclePolicyReconciler{
		Client: client,
		Scheme: scheme,
		name:   "SnapshotLifecyclePolicy",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotlifecyclepolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotlifecyclepolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=snapshotlifecyclepolicies/finalizers,verbs=update
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
func (r *SnapshotLifecyclePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, SnapshotLifecyclePolicyFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, slm, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotLifecyclePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.SnapshotLifecyclePolicy{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *SnapshotLifecyclePolicyReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	// Init condition status if not exist
	if condition.FindStatusCondition(slm.Status.Conditions, SnapshotLifecyclePolicyCondition) == nil {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:   SnapshotLifecyclePolicyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(slm.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, slm, slm.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil && slm.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current SLM policy
func (r *SnapshotLifecyclePolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if slm.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read SLM policy from Elasticsearch
	slmPolicy, err := esHandler.SLMGet(slm.GetSnapshotLifecyclePolicyName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get SLM policy from Elasticsearch")
	}
	data["current"] = slmPolicy

	// Generate expected
	expectedPolicy, err := BuildSnapshotLifecyclePolicy(slm)
	if err != nil {
		return res, errors.Wrap(err, "Unable to build SLM policy")
	}
	data["expected"] = expectedPolicy

	return res, nil
}

// Create add new SLM policy
func (r *SnapshotLifecyclePolicyReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)
	policy, err := BuildSnapshotLifecyclePolicy(slm)
	if err != nil {
		return res, errors.Wrap(err, "Error when build SnapshotLifecyclePolicy")
	}

	// Before create policy, check if repository already exist
	repo, err := esHandler.SnapshotRepositoryGet(slm.Spec.Repository)
	if err != nil {
		return res, errors.Wrap(err, "Error when get snapshot repository to check if exist before create SLM policy")
	}
	if repo == nil {
		r.log.Warnf("Snapshot repository %s not yet exist, skip it", slm.Spec.Repository)
		r.recorder.Eventf(resource, core.EventTypeWarning, "Skip", "Snapshot repository %s not yet exist, wait it", policy.Repository)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Create policy on Elasticsearch
	if err = esHandler.SLMUpdate(slm.GetSnapshotLifecyclePolicyName(), policy); err != nil {
		return res, errors.Wrap(err, "Error when update policy")
	}

	return res, nil
}

// Update permit to update current SLM policy from Elasticsearch
func (r *SnapshotLifecyclePolicyReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete SLM policy from Elasticsearch
func (r *SnapshotLifecyclePolicyReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	if err = esHandler.SLMDelete(slm.GetSnapshotLifecyclePolicyName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch SLM policy")
	}

	return nil

}

// Diff permit to check if diff between actual and expected SLM policy exist
func (r *SnapshotLifecyclePolicyReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentPolicy := d.(*eshandler.SnapshotLifecyclePolicySpec)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedPolicy := d.(*eshandler.SnapshotLifecyclePolicySpec)

	var originalPolicy *eshandler.SnapshotLifecyclePolicySpec
	if slm.Status.OriginalObject != "" {
		originalPolicy = &eshandler.SnapshotLifecyclePolicySpec{}
		if err = localhelper.UnZipBase64Decode(slm.Status.OriginalObject, originalPolicy); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentPolicy == nil {
		diff.NeedCreate = true
		diff.Diff = "SLM policy not exist"

		if err = localhelper.SetLastOriginal(slm, expectedPolicy); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.SLMDiff(currentPolicy, expectedPolicy, originalPolicy)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(slm, expectedPolicy); err != nil {
			return diff, err
		}

		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *SnapshotLifecyclePolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
		Type:    SnapshotLifecyclePolicyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	slm.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *SnapshotLifecyclePolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	slm := resource.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

	slm.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(slm.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:    SnapshotLifecyclePolicyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "SLM policy successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "SLM policy successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:    SnapshotLifecyclePolicyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "SLM policy successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "SLM policy successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(slm.Status.Conditions, SnapshotLifecyclePolicyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&slm.Status.Conditions, metav1.Condition{
			Type:    SnapshotLifecyclePolicyCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "SLM policy already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "SLM policy already set")
	}

	return nil
}
