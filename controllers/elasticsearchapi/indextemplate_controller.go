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
	IndexTemplateFinalizer = "indextemplate.elasticsearchapi.k8s.webcenter.fr/finalizer"
	IndexTemplateCondition = "IndexTemplate"
)

// IndexTemplateReconciler reconciles a index template object
type IndexTemplateReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewIndexTemplateReconciler(client client.Client, scheme *runtime.Scheme) *IndexTemplateReconciler {

	r := &IndexTemplateReconciler{
		Client: client,
		Scheme: scheme,
		name:   "indexTemplate",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indextemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indextemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=indextemplates/finalizers,verbs=update
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
func (r *IndexTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, IndexTemplateFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	it := &elasticsearchapicrd.IndexTemplate{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, it, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.IndexTemplate{}).
		Complete(r)
}

// Configure permit to init Elasticsearch handler
func (r *IndexTemplateReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	// Init condition status if not exist
	if condition.FindStatusCondition(it.Status.Conditions, IndexTemplateCondition) == nil {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:   IndexTemplateCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(it.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, it, it.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil && it.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current index template
func (r *IndexTemplateReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if it.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read index template from Elasticsearch
	currentTemplate, err := esHandler.IndexTemplateGet(it.GetIndexTemplateName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get index template from Elasticsearch")
	}
	data["current"] = currentTemplate

	// Generate expected
	expectedTemplate, err := BuildIndexTemplate(it)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate index template")
	}
	data["expected"] = expectedTemplate

	return res, nil
}

// Create add new index template
func (r *IndexTemplateReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	// Create index template on Elasticsearch
	expectedTemplate, err := BuildIndexTemplate(it)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to index template")
	}
	if err = esHandler.IndexTemplateUpdate(it.GetIndexTemplateName(), expectedTemplate); err != nil {
		return res, errors.Wrap(err, "Error when update index template")
	}

	return res, nil
}

// Update permit to update current index template from Elasticsearch
func (r *IndexTemplateReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete index template from Elasticsearch
func (r *IndexTemplateReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	if err = esHandler.IndexTemplateDelete(it.GetIndexTemplateName()); err != nil {
		return errors.Wrap(err, "Error when delete elasticsearch index template")
	}

	return nil

}

// Diff permit to check if diff between actual and expected index template exist
func (r *IndexTemplateReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	it := resource.(*elasticsearchapicrd.IndexTemplate)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentTemplate := d.(*olivere.IndicesGetIndexTemplate)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedTemplate := d.(*olivere.IndicesGetIndexTemplate)

	var originalTemplate *olivere.IndicesGetIndexTemplate
	if it.Status.OriginalObject != "" {
		originalTemplate = &olivere.IndicesGetIndexTemplate{}
		if err = localhelper.UnZipBase64Decode(it.Status.OriginalObject, originalTemplate); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentTemplate == nil {
		diff.NeedCreate = true
		diff.Diff = "Index template not exist"

		if err = localhelper.SetLastOriginal(it, expectedTemplate); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := esHandler.IndexTemplateDiff(currentTemplate, expectedTemplate, originalTemplate)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(it, expectedTemplate); err != nil {
			return diff, err
		}
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *IndexTemplateReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
		Type:    IndexTemplateCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	it.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *IndexTemplateReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	it := resource.(*elasticsearchapicrd.IndexTemplate)

	it.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(it.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:    IndexTemplateCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Index template successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Index template successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:    IndexTemplateCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Index template successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Index template successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(it.Status.Conditions, IndexTemplateCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&it.Status.Conditions, metav1.Condition{
			Type:    IndexTemplateCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Index template already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Index template already set")
	}

	return nil
}
