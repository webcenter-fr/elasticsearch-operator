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
	"time"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1alpha1"
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
	RoleFinalizer = "role.kibanaapi.k8s.webcenter.fr/finalizer"
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

//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=roles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=roles/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get
//+kubebuilder:rbac:groups="kibana.k8s.webcenter.fr",resources=kibanas,verbs=get

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

	role := &kibanaapicrd.Role{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, role, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanaapicrd.Role{}).
		Complete(r)
}

// Configure permit to init Kibana handler
func (r *RoleReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	role := resource.(*kibanaapicrd.Role)

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

	// Get Kibana handler / client
	meta, err = GetKibanaHandler(ctx, role, role.Spec.KibanaRef, r.Client, r.log)
	if err != nil && role.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init kibana handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current role
func (r *RoleReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	role := resource.(*kibanaapicrd.Role)

	// kbHandler can be empty when Kibana not yet ready or Kibana is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if role.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)

	// Read role
	currentRole, err := kbHandler.RoleGet(role.GetRoleName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get role from Kibana")
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

	kbHandler := meta.(kbhandler.KibanaHandler)
	role := resource.(*kibanaapicrd.Role)

	// Create role on Kibana
	expectedRole, err := BuildRole(role)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to Kibana role")
	}
	if err = kbHandler.RoleUpdate(expectedRole); err != nil {
		return res, errors.Wrap(err, "Error when update Kibana role")
	}

	return res, nil
}

// Update permit to update current role from Kibana
func (r *RoleReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete role from Kibana
func (r *RoleReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)
	role := resource.(*kibanaapicrd.Role)

	if err = kbHandler.RoleDelete(role.GetRoleName()); err != nil {
		return errors.Wrap(err, "Error when delete Kibana role")
	}

	return nil

}

// Diff permit to check if diff between actual and expected role exist
func (r *RoleReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	kbHandler := meta.(kbhandler.KibanaHandler)
	role := resource.(*kibanaapicrd.Role)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentRole := d.(*kbapi.KibanaRole)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedRole := d.(*kbapi.KibanaRole)

	var originalRole *kbapi.KibanaRole
	if role.Status.OriginalObject != "" {
		originalRole = &kbapi.KibanaRole{}
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
		diff.Diff = "Kibana role not exist"

		if err = localhelper.SetLastOriginal(role, expectedRole); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := kbHandler.RoleDiff(currentRole, expectedRole, originalRole)
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
	role := resource.(*kibanaapicrd.Role)

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
	role := resource.(*kibanaapicrd.Role)

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
