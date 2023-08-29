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
	RoleFinalizer = "role.elasticsearchapi.k8s.webcenter.fr/finalizer"
	RoleCondition = "Role"
)

// RoleReconciler reconciles a Role object
type RoleReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewRoleReconciler(client client.Client, scheme *runtime.Scheme) *RoleReconciler {

	r := &RoleReconciler{
		Client: client,
		Scheme: scheme,
		name:   "role",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=roles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=roles/finalizers,verbs=update
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
func (r *RoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, RoleFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	role := &elasticsearchapicrd.Role{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, role, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.Role{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *RoleReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	role := resource.(*elasticsearchapicrd.Role)

	// Init condition status if not exist
	if condition.FindStatusCondition(role.Status.Conditions, RoleCondition) == nil {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:   RoleCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(role.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, role, role.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil && role.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current role
func (r *RoleReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	role := resource.(*elasticsearchapicrd.Role)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if role.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read role
	currentRole, err := esHandler.RoleGet(role.GetRoleName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get role from Elasticsearch")
	}
	data["current"] = currentRole

	// Generate expected
	expectedRole, err := BuildRole(role)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate role")
	}
	data["expected"] = expectedRole

	return res, nil
}

// Create add new role
func (r *RoleReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	role := resource.(*elasticsearchapicrd.Role)

	// Create role on Elasticsearch
	expectedRole, err := BuildRole(role)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to elasticsearch role")
	}
	if err = esHandler.RoleUpdate(role.GetRoleName(), expectedRole); err != nil {
		return res, errors.Wrap(err, "Error when update elasticsearch role")
	}

	return res, nil
}

// Update permit to update current role from Elasticsearch
func (r *RoleReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete role from Elasticsearch
func (r *RoleReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	role := resource.(*elasticsearchapicrd.Role)

	if err = esHandler.RoleDelete(role.GetRoleName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch role")
	}

	return nil

}

// Diff permit to check if diff between actual and expected role exist
func (r *RoleReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	role := resource.(*elasticsearchapicrd.Role)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentRole := d.(*eshandler.XPackSecurityRole)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedRole := d.(*eshandler.XPackSecurityRole)

	var originalRole *eshandler.XPackSecurityRole
	if role.Status.OriginalObject != "" {
		originalRole = &eshandler.XPackSecurityRole{}
		if err = localhelper.UnZipBase64Decode(role.Status.OriginalObject, originalRole); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentRole == nil {
		diff.NeedCreate = true
		diff.Diff = "Elasticsearch role not exist"

		if err = localhelper.SetLastOriginal(role, expectedRole); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.RoleDiff(currentRole, expectedRole, originalRole)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(role, expectedRole); err != nil {
			return diff, err
		}
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *RoleReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	role := resource.(*elasticsearchapicrd.Role)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
		Type:    RoleCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	role.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *RoleReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	role := resource.(*elasticsearchapicrd.Role)

	role.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:    RoleCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Role successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Role successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:    RoleCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Role successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Role successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, RoleCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&role.Status.Conditions, metav1.Condition{
			Type:    RoleCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Role already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Role already set")
	}

	return nil
}
