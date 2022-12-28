package elasticsearch

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	o := resource.(*elasticsearchapi.Elasticsearch)

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
	o := resource.(*elasticsearchapi.Elasticsearch)
	cmList := &corev1.ConfigMapList{}

	// Read current node group configmaps
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read config maps")
	}
	data["currentConfigmaps"] = cmList.Items

	// Generate expected node group configmaps
	expectedCms, err := BuildConfigMaps(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate config maps")
	}
	data["expectedConfigmaps"] = expectedCms

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
	o := resource.(*elasticsearchapi.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentConfigmaps")
	if err != nil {
		return diff, res, err
	}
	currentCms := d.([]corev1.ConfigMap)

	d, err = helper.Get(data, "expectedConfigmaps")
	if err != nil {
		return diff, res, err
	}
	expectedCms := d.([]corev1.ConfigMap)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	cmToUpdate := make([]corev1.ConfigMap, 0)
	cmToCreate := make([]corev1.ConfigMap, 0)

	for _, expectedCm := range expectedCms {
		isFound := false
		for i, currentCm := range currentCms {
			// Need compare configmap
			if expectedCm.Name == currentCm.Name {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentCm, &expectedCm, patch.CleanMetadata(), patch.IgnoreStatusFields())
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

				// Remove items found
				currentCms = helperdiff.DeleteItemFromSlice(currentCms, i).([]corev1.ConfigMap)

				break
			}
		}

		if !isFound {
			// Need create configmap
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Configmap %s not yet exist", expectedCm.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, &expectedCm, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&expectedCm); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on configMap %s", expectedCm.Name)
			}

			cmToCreate = append(cmToCreate, expectedCm)

			r.Log.Debugf("Need create configmap %s", expectedCm.Name)
		}
	}

	if len(currentCms) > 0 {
		diff.NeedDelete = true
		for _, cm := range currentCms {
			diff.Diff.WriteString(fmt.Sprintf("Need delete configmap %s\n", cm.Name))
		}
	}

	data["newConfigmaps"] = cmToCreate
	data["configmaps"] = cmToUpdate
	data["oldConfigmaps"] = currentCms

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *ConfigMapReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

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
	o := resource.(*elasticsearchapi.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Configmaps successfully updated")
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
