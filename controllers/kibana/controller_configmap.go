package kibana

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
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
	ConfigmapCondition = "ConfigmapReady"
	ConfigmapPhase     = "Configmap"
)

type ConfigMapReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewConfiMapReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &ConfigMapReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "configmap",
	}
}

// Name return the current phase
func (r *ConfigMapReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *ConfigMapReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanaapi.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ConfigmapCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ConfigmapCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = ConfigmapPhase
	}

	return res, nil
}

// Read existing configmaps
func (r *ConfigMapReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*kibanaapi.Kibana)
	cm := &corev1.ConfigMap{}

	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetConfigMapName(o)}, cm); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read config maps")
		}
		cm = nil
	}

	data["currentConfigmap"] = cm

	// Generate expected node group configmaps
	expectedCm, err := BuildConfigMap(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate config maps")
	}
	data["expectedConfigmap"] = expectedCm

	return res, nil
}

// Create will create configmaps
func (r *ConfigMapReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newConfigmaps")
	if err != nil {
		return res, err
	}
	expectedConfigmaps := d.([]corev1.ConfigMap)

	for _, cm := range expectedConfigmaps {
		if err = r.Client.Create(ctx, &cm); err != nil {
			return res, errors.Wrapf(err, "Error when create configMap %s", cm.Name)
		}
	}

	return res, nil
}

// Update will update configmaps
func (r *ConfigMapReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "configmaps")
	if err != nil {
		return res, err
	}
	expectedConfigmaps := d.([]corev1.ConfigMap)

	for _, cm := range expectedConfigmaps {
		if err = r.Client.Update(ctx, &cm); err != nil {
			return res, errors.Wrapf(err, "Error when update configMap %s", cm.Name)
		}
	}

	return res, nil
}

// Delete permit to delete configmaps
func (r *ConfigMapReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldConfigmaps")
	if err != nil {
		return res, err
	}
	oldConfigmaps := d.([]corev1.ConfigMap)

	for _, cm := range oldConfigmaps {
		if err = r.Client.Delete(ctx, &cm); err != nil {
			return res, errors.Wrapf(err, "Error when delete configMap %s", cm.Name)
		}
	}

	return res, nil
}

// Diff permit to check if configmaps are up to date
func (r *ConfigMapReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*kibanaapi.Kibana)
	var d any

	d, err = helper.Get(data, "currentConfigmap")
	if err != nil {
		return diff, res, err
	}
	currentCm := d.(*corev1.ConfigMap)

	d, err = helper.Get(data, "expectedConfigmap")
	if err != nil {
		return diff, res, err
	}
	expectedCm := d.(*corev1.ConfigMap)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	cmToUpdate := make([]corev1.ConfigMap, 0)
	cmToCreate := make([]corev1.ConfigMap, 0)
	cmToDelete := make([]corev1.ConfigMap, 0)

	// Configmap not yet exist
	if currentCm == nil {
		diff.NeedCreate = true
		diff.Diff.WriteString(fmt.Sprintf("Configmap %s not yet exist", expectedCm.Name))

		// Set owner
		err = ctrl.SetControllerReference(o, expectedCm, r.Scheme)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference")
		}

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedCm); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on configMap %s", expectedCm.Name)
		}

		cmToCreate = append(cmToCreate, *expectedCm)

		r.Log.Debugf("Need create configmap %s", expectedCm.Name)

		data["newConfigmaps"] = cmToCreate
		data["configmaps"] = cmToUpdate
		data["oldConfigmaps"] = cmToDelete

		return diff, res, nil
	}

	// Check if is up to date
	patchResult, err := patch.DefaultPatchMaker.Calculate(currentCm, expectedCm, patch.CleanMetadata(), patch.IgnoreStatusFields())
	if err != nil {
		return diff, res, errors.Wrapf(err, "Error when diffing configmap %s", currentCm.Name)
	}
	if !patchResult.IsEmpty() {
		updatedCm := patchResult.Patched.(*corev1.ConfigMap)
		diff.NeedUpdate = true
		diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedCm.Name, string(patchResult.Patch)))
		cmToUpdate = append(cmToUpdate, *updatedCm)
		r.Log.Debugf("Need update configmap %s", updatedCm.Name)
	}

	data["newConfigmaps"] = cmToCreate
	data["configmaps"] = cmToUpdate
	data["oldConfigmaps"] = cmToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *ConfigMapReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*kibanaapi.Kibana)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    ConfigmapCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ConfigMapReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*kibanaapi.Kibana)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Configmap successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ConfigmapCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    ConfigmapCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
