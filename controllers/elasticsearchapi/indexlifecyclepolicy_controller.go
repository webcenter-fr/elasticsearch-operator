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
	core "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IndexLifecyclePolicyFinalizer = "ilm.elasticsearchapi.k8s.webcenter.fr/finalizer"
	IndexLifecyclePolicyCondition = "IndexLifecyclePolicy"
)

// IndexLifecyclePolicyReconciler reconciles a IndexLifecyclePolicy object
type IndexLifecyclePolicyReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewIndexLifecyclePolicyReconciler(client client.Client, scheme *runtime.Scheme) *IndexLifecyclePolicyReconciler {

	r := &IndexLifecyclePolicyReconciler{
		Client: client,
		Scheme: scheme,
		name:   "indexLifecyclePolicy",
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

	reconciler, err := controller.NewStdReconciler(r.Client, IndexLifecyclePolicyFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	ilm := &elasticsearchapicrd.IndexLifecyclePolicy{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, ilm, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexLifecyclePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.IndexLifecyclePolicy{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *IndexLifecyclePolicyReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)

	// Init condition status if not exist
	if condition.FindStatusCondition(ilm.Status.Conditions, IndexLifecyclePolicyCondition) == nil {
		condition.SetStatusCondition(&ilm.Status.Conditions, metav1.Condition{
			Type:   IndexLifecyclePolicyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, ilm.Spec.ElasticsearchRef, r.Client, req, r.log)
	if err != nil {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, err
}

// Read permit to get current ILM policy
func (r *IndexLifecyclePolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if ilm.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read ILM policy from Elasticsearch
	ilmPolicy, err := esHandler.ILMGet(ilm.Name)
	if err != nil {
		return res, errors.Wrap(err, "Unable to get ILM policy from Elasticsearch")
	}

	data["policy"] = ilmPolicy

	return res, nil
}

// Create add new ILM policy
func (r *IndexLifecyclePolicyReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)
	policy := &olivere.XPackIlmGetLifecycleResponse{}

	// Create policy on Elasticsearch
	if err = json.Unmarshal([]byte(ilm.Spec.Policy), &policy); err != nil {
		return res, errors.Wrap(err, "Error on Policy format")
	}
	if err = esHandler.ILMUpdate(ilm.Name, policy); err != nil {
		return res, errors.Wrap(err, "Error when update policy")
	}

	return res, nil
}

// Update permit to update current ILM policy from Elasticsearch
func (r *IndexLifecyclePolicyReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete ILM policy from Elasticsearch
func (r *IndexLifecyclePolicyReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)

	if err = esHandler.ILMDelete(ilm.Name); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch ILM policy")
	}

	return nil

}

// Diff permit to check if diff between actual and expected ILM policy exist
func (r *IndexLifecyclePolicyReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)
	expectedPolicy := &olivere.XPackIlmGetLifecycleResponse{}
	var currentPolicy *olivere.XPackIlmGetLifecycleResponse
	var d any

	d, err = helper.Get(data, "policy")
	if err != nil {
		return diff, err
	}
	currentPolicy = d.(*olivere.XPackIlmGetLifecycleResponse)
	if err = json.Unmarshal([]byte(ilm.Spec.Policy), expectedPolicy); err != nil {
		return diff, err
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentPolicy == nil {
		diff.NeedCreate = true
		diff.Diff = "ILM policy not exist"
		return diff, nil
	}

	diffStr, err := esHandler.ILMDiff(currentPolicy, expectedPolicy)
	if err != nil {
		return diff, err
	}

	if diffStr != "" {
		diff.NeedUpdate = true
		diff.Diff = diffStr
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *IndexLifecyclePolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&ilm.Status.Conditions, metav1.Condition{
		Type:    IndexLifecyclePolicyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	ilm.Status.Health = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *IndexLifecyclePolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	ilm := resource.(*elasticsearchapicrd.IndexLifecyclePolicy)

	ilm.Status.Health = true

	if diff.NeedCreate {
		condition.SetStatusCondition(&ilm.Status.Conditions, metav1.Condition{
			Type:    IndexLifecyclePolicyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "ILM policy successfully created",
		})

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&ilm.Status.Conditions, metav1.Condition{
			Type:    IndexLifecyclePolicyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "ILM policy successfully updated",
		})

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(ilm.Status.Conditions, IndexLifecyclePolicyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&ilm.Status.Conditions, metav1.Condition{
			Type:    IndexLifecyclePolicyCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "ILM policy already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "ILM policy already set")
	}

	return nil
}
