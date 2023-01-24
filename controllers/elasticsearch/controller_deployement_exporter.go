package elasticsearch

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ExporterCondition = "ExporterReady"
	ExporterPhase     = "Exporter"
)

type ExporterReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewExporterReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &ExporterReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "exporter",
	}
}

// Name return the current phase
func (r *ExporterReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *ExporterReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ExporterCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ExporterCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = ExporterPhase
	}

	return res, nil
}

// Read existing deployement
func (r *ExporterReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	dpl := &appv1.Deployment{}

	// Read current deployment
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetExporterDeployementName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read exporter deployment")
		}
		dpl = nil
	}
	data["currentExporter"] = dpl

	// Generate expected deployement
	expectedExporter, err := BuildDeploymentExporter(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate exporter deployment")
	}
	data["expectedExporter"] = expectedExporter

	return res, nil
}

// Create will create deployment
func (r *ExporterReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newExporters")
	if err != nil {
		return res, err
	}
	expectedExporters := d.([]appv1.Deployment)

	for _, dpl := range expectedExporters {
		if err = r.Client.Create(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when create exporter deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Update will update deployment
func (r *ExporterReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "exporters")
	if err != nil {
		return res, err
	}
	expectedExporters := d.([]appv1.Deployment)

	for _, dpl := range expectedExporters {
		if err = r.Client.Update(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when update exporter deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Delete permit to delete deployment
func (r *ExporterReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldExporters")
	if err != nil {
		return res, err
	}
	oldExporters := d.([]appv1.Deployment)

	for _, dpl := range oldExporters {
		if err = r.Client.Delete(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when delete exporter deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Diff permit to check if deployment is up to date
func (r *ExporterReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentExporter")
	if err != nil {
		return diff, res, err
	}
	currentExporter := d.(*appv1.Deployment)

	d, err = helper.Get(data, "expectedExporter")
	if err != nil {
		return diff, res, err
	}
	expectedExporter := d.(*appv1.Deployment)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	exporterToUpdate := make([]appv1.Deployment, 0)
	exporterToCreate := make([]appv1.Deployment, 0)
	exporterToDelete := make([]appv1.Deployment, 0)

	if currentExporter == nil {
		if expectedExporter != nil {
			// Create new exporter
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create exporter deployment %s\n", expectedExporter.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, expectedExporter, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedExporter); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on exporter deployment %s", expectedExporter.Name)
			}

			exporterToCreate = append(exporterToCreate, *expectedExporter)
		}
	} else {

		if expectedExporter != nil {
			// Check if need to update deployment
			patchResult, err := patch.DefaultPatchMaker.Calculate(currentExporter, expectedExporter, patch.CleanMetadata(), patch.IgnoreStatusFields())
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when diffing exporter deployment %s", currentExporter.Name)
			}
			if !patchResult.IsEmpty() {
				updatedExporter := patchResult.Patched.(*appv1.Deployment)
				diff.NeedUpdate = true
				diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedExporter.Name, string(patchResult.Patch)))
				exporterToUpdate = append(exporterToUpdate, *updatedExporter)
				r.Log.Debugf("Need update exporter %s", updatedExporter.Name)
			}
		} else {
			// Need delete deployment
			diff.NeedDelete = true
			diff.Diff.WriteString(fmt.Sprintf("Delete exporter deployment %s\n", currentExporter.Name))
			exporterToDelete = append(exporterToDelete, *currentExporter)
		}

	}

	data["newExporters"] = exporterToCreate
	data["exporters"] = exporterToUpdate
	data["oldExporters"] = exporterToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *ExporterReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    ExporterCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ExporterReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Exporter deployment successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ExporterCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    ExporterCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
