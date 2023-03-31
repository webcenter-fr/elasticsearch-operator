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
	LogstashPipelineFinalizer = "pipeline.kibanaapi.k8s.webcenter.fr/finalizer"
	LogstashPipelineCondition = "LogstashPipeline"
)

// LogstashPipelineReconciler reconciles a Logstash pipeline object
type LogstashPipelineReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewLogstashPipelineReconciler(client client.Client, scheme *runtime.Scheme) *LogstashPipelineReconciler {

	r := &LogstashPipelineReconciler{
		Client: client,
		Scheme: scheme,
		name:   "logstashPipeline",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get
//+kubebuilder:rbac:groups="kibana.k8s.webcenter.fr",resources=kibanas,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Logstash Pipeline object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LogstashPipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, LogstashPipelineFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	pipeline := &kibanaapicrd.LogstashPipeline{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, pipeline, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LogstashPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanaapicrd.LogstashPipeline{}).
		Complete(r)
}

// Configure permit to init Kibana handler
func (r *LogstashPipelineReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	// Init condition status if not exist
	if condition.FindStatusCondition(pipeline.Status.Conditions, LogstashPipelineCondition) == nil {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:   LogstashPipelineCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(pipeline.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get Kibana handler / client
	meta, err = GetKibanaHandler(ctx, pipeline, pipeline.Spec.KibanaRef, r.Client, r.log)
	if err != nil && pipeline.DeletionTimestamp.IsZero() {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init kibana handler: %s", err.Error())
		return nil, err
	}

	return meta, nil
}

// Read permit to get current pipeline
func (r *LogstashPipelineReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	// kbHandler can be empty when Kibana not yet ready or Kibana is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if pipeline.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)

	// Read pipeline
	currentPipeline, err := kbHandler.LogstashPipelineGet(pipeline.GetPipelineName())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get logstash pipeline from Kibana")
	}
	data["current"] = currentPipeline

	// Generate expected
	expectedPipeline, err := BuildLogstashPipeline(pipeline)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate logstash pipeline")
	}
	data["expected"] = expectedPipeline

	return res, nil
}

// Create add new pipeline
func (r *LogstashPipelineReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	kbHandler := meta.(kbhandler.KibanaHandler)
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	// Create pipeline on Kibana
	expectedPipeline, err := BuildLogstashPipeline(pipeline)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert to Kibana logstahs pipeline")
	}
	if err = kbHandler.LogstashPipelineUpdate(expectedPipeline); err != nil {
		return res, errors.Wrap(err, "Error when update Kibana logstash pipeline")
	}

	return res, nil
}

// Update permit to update current pipeline from Kibana
func (r *LogstashPipelineReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete pipeline from Kibana
func (r *LogstashPipelineReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	kbHandler := meta.(kbhandler.KibanaHandler)
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	if err = kbHandler.LogstashPipelineDelete(pipeline.GetPipelineName()); err != nil {
		return errors.Wrap(err, "Error when delete Kibana logstash pipeline")
	}

	return nil

}

// Diff permit to check if diff between actual and expected logstash pipeline exist
func (r *LogstashPipelineReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	kbHandler := meta.(kbhandler.KibanaHandler)
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)
	var d any

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentPipeline := d.(*kbapi.LogstashPipeline)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedPipeline := d.(*kbapi.LogstashPipeline)

	var originalPipeline *kbapi.LogstashPipeline
	if pipeline.Status.OriginalObject != "" {
		originalPipeline = &kbapi.LogstashPipeline{}
		if err = localhelper.UnZipBase64Decode(pipeline.Status.OriginalObject, originalPipeline); err != nil {
			return diff, err
		}
	}

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}

	if currentPipeline == nil {
		diff.NeedCreate = true
		diff.Diff = "Kibana logstash Pipeline not exist"

		if err = localhelper.SetLastOriginal(pipeline, expectedPipeline); err != nil {
			return diff, err
		}

		return diff, nil
	}

	differ, err := kbHandler.LogstashPipelineDiff(currentPipeline, expectedPipeline, originalPipeline)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(pipeline, expectedPipeline); err != nil {
			return diff, err
		}
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *LogstashPipelineReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
		Type:    LogstashPipelineCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	pipeline.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LogstashPipelineReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	pipeline := resource.(*kibanaapicrd.LogstashPipeline)

	pipeline.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:    LogstashPipelineCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Logstash Pipeline successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Logstash pipeline successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:    LogstashPipelineCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "Logstash pipeline successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Logstash pipeline successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, LogstashPipelineCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
			Type:    LogstashPipelineCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Logstash pipeline already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Logstash pipeline already set")
	}

	return nil
}
