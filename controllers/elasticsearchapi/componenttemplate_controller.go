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
	ComponentTemplateFinalizer = "componenttemplate.elasticsearchapi.k8s.webcenter.fr/finalizer"
	ComponentTemplateCondition = "ComponentTemplate"
)

// ComponentTemplateReconciler reconciles a component template object
type ComponentTemplateReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewComponentTemplateReconciler(client client.Client, scheme *runtime.Scheme) *ComponentTemplateReconciler {

	r := &ComponentTemplateReconciler{
		Client: client,
		Scheme: scheme,
		name:   "componentTemplate",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=componenttemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=componenttemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=componenttemplates/finalizers,verbs=update
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
func (r *ComponentTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, ComponentTemplateFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	ct := &elasticsearchapicrd.ComponentTemplate{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, ct, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComponentTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.ComponentTemplate{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *ComponentTemplateReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	// Init condition status if not exist
	if condition.FindStatusCondition(ct.Status.Conditions, ComponentTemplateCondition) == nil {
		condition.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
			Type:   ComponentTemplateCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, ct.Spec.ElasticsearchRef, r.Client, req, r.log)
	if err != nil {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, err
}

// Read permit to get current component template
func (r *ComponentTemplateReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if ct.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read component template from Elasticsearch
	currentComponent, err := esHandler.ComponentTemplateGet(ct.GetComponentTemplateName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get component template from Elasticsearch")
	}

	data["component"] = currentComponent

	return res, nil
}

// Create add new component template
func (r *ComponentTemplateReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	// Create policy on Elasticsearch
	expectedComponent, err := BuildComponentTemplate(ct)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert current component template to expected component template")
	}

	if err = esHandler.ComponentTemplateUpdate(ct.GetComponentTemplateName(), expectedComponent); err != nil {
		return res, errors.Wrap(err, "Error when update component template")
	}

	return res, nil
}

// Update permit to update current component template from Elasticsearch
func (r *ComponentTemplateReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete component template from Elasticsearch
func (r *ComponentTemplateReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	if err = esHandler.ComponentTemplateDelete(ct.GetComponentTemplateName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch component template")
	}

	return nil

}

// Diff permit to check if diff between actual and expected component template exist
func (r *ComponentTemplateReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	var currentComponent *olivere.IndicesGetComponentTemplate
	var d any

	d, err = helper.Get(data, "component")
	if err != nil {
		return diff, err
	}
	currentComponent = d.(*olivere.IndicesGetComponentTemplate)
	expectedComponent, err := BuildComponentTemplate(ct)
	if err != nil {
		return diff, err
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentComponent == nil {
		diff.NeedCreate = true
		diff.Diff = "Component template not exist"
		return diff, nil
	}

	diffStr, err := esHandler.ComponentTemplateDiff(currentComponent, expectedComponent)
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
func (r *ComponentTemplateReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
		Type:    ComponentTemplateCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	ct.Status.Health = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ComponentTemplateReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	ct := resource.(*elasticsearchapicrd.ComponentTemplate)

	ct.Status.Health = true

	if diff.NeedCreate {
		condition.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
			Type:    ComponentTemplateCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Component template successfully created",
		})

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
			Type:    ComponentTemplateCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Component template successfully updated",
		})

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(ct.Status.Conditions, ComponentTemplateCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&ct.Status.Conditions, metav1.Condition{
			Type:    ComponentTemplateCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Component template already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Component template already set")
	}

	return nil
}
