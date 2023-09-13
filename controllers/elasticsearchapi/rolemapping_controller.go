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
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	core "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RoleMappingFinalizer = "rolemapping.elasticsearchapi.k8s.webcenter.fr/finalizer"
	RoleMappingCondition = "RoleMapping"
)

// RoleMappingReconciler reconciles a RoleMapping object
type RoleMappingReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewRoleMappingReconciler(client client.Client, scheme *runtime.Scheme) *RoleMappingReconciler {

	r := &RoleMappingReconciler{
		Client: client,
		Scheme: scheme,
		name:   "roleMapping",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=rolemappings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=rolemappings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=rolemappings/finalizers,verbs=update
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
func (r *RoleMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, RoleMappingFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	rm := &elasticsearchapicrd.RoleMapping{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, rm, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.RoleMapping{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *RoleMappingReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	// Init condition status if not exist
	if condition.FindStatusCondition(rm.Status.Conditions, RoleMappingCondition) == nil {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:   RoleMappingCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(rm.Status.Conditions, common.ReadyCondition.String()) == nil {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, rm, rm.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil && rm.DeletionTimestamp.IsZero() {
		return nil, err
	}

	return meta, nil
}

// Read permit to get current roleMapping
func (r *RoleMappingReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if rm.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read role mapping from Elasticsearch
	currentRoleMapping, err := esHandler.RoleMappingGet(rm.GetRoleMappingName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get role mapping from Elasticsearch")
	}
	data["current"] = currentRoleMapping

	// Generate expected
	expectedRoleMapping, err := BuildRoleMapping(rm)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate role mapping")
	}
	data["expected"] = expectedRoleMapping

	return res, nil
}

// Create add new role mapping
func (r *RoleMappingReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	// Create role mapping on Elasticsearch
	expectedRoleMapping, err := BuildRoleMapping(rm)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to elasticsearch role mapping")
	}
	if err = esHandler.RoleMappingUpdate(rm.GetRoleMappingName(), expectedRoleMapping); err != nil {
		return res, errors.Wrap(err, "Error when update elasticsearch role mapping")
	}

	return res, nil
}

// Update permit to update current role mapping from Elasticsearch
func (r *RoleMappingReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete role mapping from Elasticsearch
func (r *RoleMappingReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	if err = esHandler.RoleMappingDelete(rm.GetRoleMappingName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch roleMapping")
	}

	return nil

}

// Diff permit to check if diff between actual and expected role mapping exist
func (r *RoleMappingReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	rm := resource.(*elasticsearchapicrd.RoleMapping)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentRoleMapping := d.(*olivere.XPackSecurityRoleMapping)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedRoleMapping := d.(*olivere.XPackSecurityRoleMapping)

	var originalRoleMapping *olivere.XPackSecurityRoleMapping
	if rm.Status.OriginalObject != "" {
		originalRoleMapping = &olivere.XPackSecurityRoleMapping{}
		if err = localhelper.UnZipBase64Decode(rm.Status.OriginalObject, originalRoleMapping); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentRoleMapping == nil {
		diff.NeedCreate = true
		diff.Diff = "Elasticsearch role mapping not exist"

		if err = localhelper.SetLastOriginal(rm, expectedRoleMapping); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.RoleMappingDiff(currentRoleMapping, expectedRoleMapping, originalRoleMapping)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(rm, expectedRoleMapping); err != nil {
			return diff, err
		}

		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *RoleMappingReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	r.log.Error(err)

	condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
		Type:    RoleMappingCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: strings.ShortenString(err.Error(), common.ShortenError),
	})

	condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition.String(),
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	rm.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *RoleMappingReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	rm := resource.(*elasticsearchapicrd.RoleMapping)

	rm.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(rm.Status.Conditions, common.ReadyCondition.String(), metav1.ConditionFalse) {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition.String(),
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:    RoleMappingCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "RoleMapping successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Role mapping successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:    RoleMappingCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "RoleMapping successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Role mapping successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(rm.Status.Conditions, RoleMappingCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&rm.Status.Conditions, metav1.Condition{
			Type:    RoleMappingCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "RoleMapping already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "RoleMapping already set")
	}

	return nil
}
