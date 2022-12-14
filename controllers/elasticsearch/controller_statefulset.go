package elasticsearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	podutil "k8s.io/kubectl/pkg/util/podutils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type statefulsetPhase string

const (
	StatefulsetCondition        = "StatefulsetReady"
	StatefulsetConditionUpgrade = "StatefulsetUpgrade"
	StatefulsetPhase            = "Statefullset"
)

var (
	phaseStsUpgrade         statefulsetPhase = "statefulsetUpgrade"
	phaseStsUpgradeFinished statefulsetPhase = "statefulsetUpgradeFinished"
	phaseStsNormal          statefulsetPhase = "statefulsetNormal"
)

type StatefulsetReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewStatefulsetReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &StatefulsetReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "statefulset",
	}
}

// Name return the current phase
func (r *StatefulsetReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *StatefulsetReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, StatefulsetCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   StatefulsetCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = StatefulsetPhase
	}

	return res, nil
}

// Read existing satefulsets
func (r *StatefulsetReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	stsList := &appv1.StatefulSetList{}

	// Read current satefulsets
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read statefulset")
	}
	data["currentStatefulsets"] = stsList.Items

	// Generate expected statefulsets
	expectedSts, err := BuildStatefulsets(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate statefulsets")
	}
	data["expectedStatefulsets"] = expectedSts

	return res, nil
}

// Create will create statefulset
func (r *StatefulsetReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newStatefulsets")
	if err != nil {
		return res, err
	}
	expectedSts := d.([]appv1.StatefulSet)

	for _, sts := range expectedSts {
		if err = r.Client.Create(ctx, &sts); err != nil {
			return res, errors.Wrapf(err, "Error when create statefulset %s", sts.Name)
		}
	}

	return res, nil
}

// Update will update statefulset
func (r *StatefulsetReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "statefulsets")
	if err != nil {
		return res, err
	}
	expectedSts := d.([]appv1.StatefulSet)

	for _, sts := range expectedSts {
		if err = r.Client.Update(ctx, &sts); err != nil {
			return res, errors.Wrapf(err, "Error when update statefulset %s", sts.Name)
		}
	}

	return res, nil
}

// Delete permit to delete statefulset
func (r *StatefulsetReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldStatefulsets")
	if err != nil {
		return res, err
	}
	oldSts := d.([]appv1.StatefulSet)

	for _, sts := range oldSts {
		if err = r.Client.Delete(ctx, &sts); err != nil {
			return res, errors.Wrapf(err, "Error when delete statefulset %s", sts.Name)
		}
	}

	return res, nil
}

// Diff permit to check if statefulset is up to date
func (r *StatefulsetReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentStatefulsets")
	if err != nil {
		return diff, res, err
	}
	currentStatefulsets := d.([]appv1.StatefulSet)

	d, err = helper.Get(data, "expectedStatefulsets")
	if err != nil {
		return diff, res, err
	}
	expectedStatefulsets := d.([]appv1.StatefulSet)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	stsToUpdate := make([]appv1.StatefulSet, 0)
	stsToCreate := make([]appv1.StatefulSet, 0)

	// Add some code to avoid reconcile multiple statefullset on same time
	// It avoid to have multiple pod that exit the cluster on same time

	// Check if on current upgrade phase, to wait the end
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseStsUpgrade)

		podList := &corev1.PodList{}
		for _, sts := range currentStatefulsets {
			if sts.Annotations[fmt.Sprintf("%s/upgrade", ElasticsearchAnnotationKey)] == "true" {
				// Check the pod state
				labelSelectors := labels.SelectorFromSet(sts.Spec.Template.Labels)
				if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
					return diff, res, errors.Wrapf(err, "Error when read Elasticsearch pods")
				}

				isFinished := true
				annotation := fmt.Sprintf("%s/upgrade-checksum", ElasticsearchAnnotationKey)
				stsChecksum := sts.Spec.Template.Annotations[annotation]
				for _, p := range podList.Items {
					// The pod must have CA checksum annotation and need to be ready
					if p.Annotations[annotation] == "" || p.Annotations[annotation] != stsChecksum || !podutil.IsPodReady(&p) {
						isFinished = false
					}
				}
				if !isFinished {
					// All Sts not yet finished upgrade
					r.Log.Info("Phase statefulset upgrade: wait pod to be ready")
					return diff, ctrl.Result{RequeueAfter: time.Second * 30}, nil
				}

				r.Log.Infof("Statefulset %s successfully finished upgrade", sts.Name)

				// Clean annotations
				sts.Annotations[fmt.Sprintf("%s/upgrade", ElasticsearchAnnotationKey)] = "false"
				stsToUpdate = append(stsToUpdate, sts)
				diff.NeedUpdate = true

				data["newStatefulsets"] = stsToCreate
				data["statefulsets"] = stsToUpdate
				data["oldStatefulsets"] = currentStatefulsets
				data["phase"] = phaseStsUpgradeFinished

				return diff, res, nil
			}
		}

	}
	for _, expectedSts := range expectedStatefulsets {
		isFound := false
		for i, currentSts := range currentStatefulsets {
			// Need compare statefulset
			if currentSts.Name == expectedSts.Name {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentSts, &expectedSts, patch.CleanMetadata(), patch.IgnoreStatusFields())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing statefulset %s", currentSts.Name)
				}
				if !patchResult.IsEmpty() {
					updatedSts := patchResult.Patched.(*appv1.StatefulSet)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedSts.Name, string(patchResult.Patch)))

					// Add annotations to wait reconcile
					updatedSts.Annotations[fmt.Sprintf("%s/upgrade", ElasticsearchAnnotationKey)] = "true"
					sum, err := checksum.SHA256sumReader(strings.NewReader(updatedSts.ResourceVersion))
					if err != nil {
						return diff, res, errors.Wrapf(err, "Error when generate checksum to track statefulset %s upgrade", updatedSts.Name)
					}
					updatedSts.Spec.Template.Annotations[fmt.Sprintf("%s/upgrade-checksum", ElasticsearchAnnotationKey)] = sum

					stsToUpdate = append(stsToUpdate, *updatedSts)
					r.Log.Debugf("Need update statefulset %s", updatedSts.Name)

					data["newStatefulsets"] = stsToCreate
					data["statefulsets"] = stsToUpdate
					data["oldStatefulsets"] = currentStatefulsets
					data["phase"] = phaseStsUpgrade

					return diff, res, nil
				}

				// Remove items found
				currentStatefulsets = helperdiff.DeleteItemFromSlice(currentStatefulsets, i).([]appv1.StatefulSet)

				break
			}
		}

		if !isFound {
			// Need create services
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Statefulset %s not yet exist\n", expectedSts.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, &expectedSts, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&expectedSts); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on statefulset %s", expectedSts.Name)
			}

			stsToCreate = append(stsToCreate, expectedSts)

			r.Log.Debugf("Need create statefulset %s", expectedSts.Name)
		}
	}

	if len(currentStatefulsets) > 0 {
		diff.NeedDelete = true
		for _, sts := range currentStatefulsets {
			diff.Diff.WriteString(fmt.Sprintf("Need delete statefulset %s\n", sts.Name))
		}
	}

	data["newStatefulsets"] = stsToCreate
	data["statefulsets"] = stsToUpdate
	data["oldStatefulsets"] = currentStatefulsets
	data["phase"] = phaseStsNormal

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *StatefulsetReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    StatefulsetCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *StatefulsetReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var (
		d any
	)

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(statefulsetPhase)

	switch phase {
	case phaseStsUpgrade:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "Statefulsets are being upgraded",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are being upgraded",
		})

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are being upgraded")

		return ctrl.Result{Requeue: true}, nil

	case phaseStsUpgradeFinished:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are finished to be upgraded",
		})

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are finished to be upgraded")

		return ctrl.Result{Requeue: true}, nil

	}

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionFalse) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "No current upgrade",
		})
	}

	return res, nil
}
