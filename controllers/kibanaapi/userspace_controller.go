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
	UserSpaceFinalizer = "space.kibanaapi.k8s.webcenter.fr/finalizer"
	UserSpaceCondition = "Space"
)

// UserSpaceReconciler reconciles a user space object
type UserSpaceReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewUserSpaceReconciler(client client.Client, scheme *runtime.Scheme) *UserSpaceReconciler {

	r := &UserSpaceReconciler{
		Client: client,
		Scheme: scheme,
		name:   "userSpace",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=userspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get
//+kubebuilder:rbac:groups="kibana.k8s.webcenter.fr",resources=kibanas,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UserSpace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *UserSpaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, UserSpaceFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	space := &kibanaapicrd.UserSpace{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, space, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserSpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanaapicrd.UserSpace{}).
		Complete(r)
}

// Configure permit to init Kibana handler
func (r *UserSpaceReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	space := resource.(*kibanaapicrd.UserSpace)

	// Init condition status if not exist
	if condition.FindStatusCondition(space.Status.Conditions, UserSpaceCondition) == nil {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:   UserSpaceCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(space.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get Kibana handler / client
	meta, err = GetKibanaHandler(ctx, space, space.Spec.KibanaRef, r.Client, r.log)
	if err != nil && space.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init kibana handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current user space
func (r *UserSpaceReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	space := resource.(*kibanaapicrd.UserSpace)

	// kbHandler can be empty when Kibana not yet ready or Kibana is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if space.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)

	// Read user space
	currentSpace, err := kbHandler.UserSpaceGet(space.GetUserSpaceID())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get user space from Kibana")
	}
	data["current"] = currentSpace

	// Generate expected
	expectedSpace, err := BuildUserSpace(space)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate user space")
	}
	data["expected"] = expectedSpace

	return res, nil
}

// Create add new user space
func (r *UserSpaceReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	kbHandler := meta.(kbhandler.KibanaHandler)
	space := resource.(*kibanaapicrd.UserSpace)

	// Create user space on Kibana
	expectedSpace, err := BuildUserSpace(space)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to Kibana user space")
	}
	if err = kbHandler.UserSpaceCreate(expectedSpace); err != nil {
		return res, errors.Wrap(err, "Error when create Kibana user space")
	}

	// Copy object that not enforce reconcile
	for _, copySpec := range space.Spec.KibanaUserSpaceCopies {
		if !copySpec.IsForceUpdate() {
			if err = kbHandler.UserSpaceCopyObject(copySpec.OriginUserSpace, &kbapi.KibanaSpaceCopySavedObjectParameter{
				Spaces:            []string{space.GetUserSpaceID()},
				IncludeReferences: copySpec.IsIncludeReference(),
				Overwrite:         copySpec.IsOverwrite(),
				CreateNewCopies:   copySpec.IsCreateNewCopy(),
			}); err != nil {
				return res, errors.Wrap(err, "Error when copy objects on new Kibana user space")
			}
		}
	}

	return res, nil
}

// Update permit to update current user space from Kibana
func (r *UserSpaceReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	kbHandler := meta.(kbhandler.KibanaHandler)
	space := resource.(*kibanaapicrd.UserSpace)

	// Create user space on Kibana
	expectedSpace, err := BuildUserSpace(space)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to Kibana user space")
	}
	if err = kbHandler.UserSpaceUpdate(expectedSpace); err != nil {
		return res, errors.Wrap(err, "Error when update Kibana user space")
	}

	return res, nil
}

// Delete permit to delete user space from Kibana
func (r *UserSpaceReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)
	space := resource.(*kibanaapicrd.UserSpace)

	if err = kbHandler.UserSpaceDelete(space.GetUserSpaceID()); err != nil {
		return errors.Wrap(err, "Error when delete Kibana user space")
	}

	return nil

}

// Diff permit to check if diff between actual and expected user space exist
func (r *UserSpaceReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	kbHandler := meta.(kbhandler.KibanaHandler)
	space := resource.(*kibanaapicrd.UserSpace)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentSpace := d.(*kbapi.KibanaSpace)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedSpace := d.(*kbapi.KibanaSpace)

	var originalSpace *kbapi.KibanaSpace
	if space.Status.OriginalObject != "" {
		originalSpace = &kbapi.KibanaSpace{}
		if err = localhelper.UnZipBase64Decode(space.Status.OriginalObject, originalSpace); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentSpace == nil {
		diff.NeedCreate = true
		diff.Diff = "Kibana user space not exist"

		if err = localhelper.SetLastOriginal(space, expectedSpace); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := kbHandler.UserSpaceDiff(currentSpace, expectedSpace, originalSpace)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(space, expectedSpace); err != nil {
			return diff, err
		}
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *UserSpaceReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	space := resource.(*kibanaapicrd.UserSpace)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
		Type:    UserSpaceCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	space.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *UserSpaceReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	space := resource.(*kibanaapicrd.UserSpace)
	kbHandler := meta.(kbhandler.KibanaHandler)

	// Copy object on force reconcile
	for _, copySpec := range space.Spec.KibanaUserSpaceCopies {
		if copySpec.IsForceUpdate() {
			if err = kbHandler.UserSpaceCopyObject(copySpec.OriginUserSpace, &kbapi.KibanaSpaceCopySavedObjectParameter{
				Spaces:            []string{space.GetUserSpaceID()},
				IncludeReferences: copySpec.IsIncludeReference(),
				Overwrite:         true,
				CreateNewCopies:   false,
			}); err != nil {
				return errors.Wrap(err, "Error when copy objects on user space")
			}
		}
	}

	space.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:    UserSpaceCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "User space successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User space successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:    UserSpaceCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "User space successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User space successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, UserSpaceCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&space.Status.Conditions, metav1.Condition{
			Type:    UserSpaceCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "User space already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User space already set")
	}

	return nil
}
