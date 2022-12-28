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
	policyv1 "k8s.io/api/policy/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PdbCondition = "PodDisruptionBudgetReady"
	PdbPhase     = "PodDisruptionBudget"
)

type PdbReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewPdbReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &PdbReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "podDisruptionBudget",
	}
}

// Name return the current phase
func (r *PdbReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *PdbReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, PdbCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   PdbCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = PdbPhase
	}

	return res, nil
}

// Read existing pdbs
func (r *PdbReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	pdbList := &policyv1.PodDisruptionBudgetList{}

	// Read current node group pdbs
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, pdbList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read service")
	}
	data["currentPdbs"] = pdbList.Items

	// Generate expected node group pdbs
	expectedPdbs, err := BuildPodDisruptionBudget(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate pdbs")
	}
	data["expectedPdbs"] = expectedPdbs

	return res, nil
}

// Create will create pdbs
func (r *PdbReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newPdbs")
	if err != nil {
		return res, err
	}
	expectedPdbs := d.([]policyv1.PodDisruptionBudget)

	for _, pdb := range expectedPdbs {
		if err = r.Client.Create(ctx, &pdb); err != nil {
			return res, errors.Wrapf(err, "Error when create pdb %s", pdb.Name)
		}
	}

	return res, nil
}

// Update will update pdbs
func (r *PdbReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "pdbs")
	if err != nil {
		return res, err
	}
	expectedPdbs := d.([]policyv1.PodDisruptionBudget)

	for _, pdb := range expectedPdbs {
		if err = r.Client.Update(ctx, &pdb); err != nil {
			return res, errors.Wrapf(err, "Error when update pdb %s", pdb.Name)
		}
	}

	return res, nil
}

// Delete permit to delete pdb
func (r *PdbReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldPdbs")
	if err != nil {
		return res, err
	}
	oldPdbs := d.([]policyv1.PodDisruptionBudget)

	for _, pdb := range oldPdbs {
		if err = r.Client.Delete(ctx, &pdb); err != nil {
			return res, errors.Wrapf(err, "Error when delete pdb %s", pdb.Name)
		}
	}

	return res, nil
}

// Diff permit to check if pdbs are up to date
func (r *PdbReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentPdbs")
	if err != nil {
		return diff, res, err
	}
	currentPdbs := d.([]policyv1.PodDisruptionBudget)

	d, err = helper.Get(data, "expectedPdbs")
	if err != nil {
		return diff, res, err
	}
	expectedPdbs := d.([]policyv1.PodDisruptionBudget)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	pdbToUpdate := make([]policyv1.PodDisruptionBudget, 0)
	pdbToCreate := make([]policyv1.PodDisruptionBudget, 0)

	for _, expectedPdb := range expectedPdbs {
		isFound := false
		for i, currentPdb := range currentPdbs {
			// Need compare pdb
			if currentPdb.Name == expectedPdb.Name {
				isFound = true

				// Copy TypeMeta to work with IgnorePDBSelector()
				expectedPdb.TypeMeta = currentPdb.TypeMeta
				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentPdb, &expectedPdb, patch.IgnorePDBSelector(), patch.CleanMetadata(), patch.IgnoreStatusFields())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing pdb %s", currentPdb.Name)
				}
				if !patchResult.IsEmpty() {
					updatedPdb := patchResult.Patched.(*policyv1.PodDisruptionBudget)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedPdb.Name, string(patchResult.Patch)))
					pdbToUpdate = append(pdbToUpdate, *updatedPdb)
					r.Log.Debugf("Need update pdb %s", updatedPdb.Name)
				}

				// Remove items found
				currentPdbs = helperdiff.DeleteItemFromSlice(currentPdbs, i).([]policyv1.PodDisruptionBudget)

				break
			}
		}

		if !isFound {
			// Need create pdbs
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Pdb %s not yet exist\n", expectedPdb.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, &expectedPdb, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&expectedPdb); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on pdb %s", expectedPdb.Name)
			}

			pdbToCreate = append(pdbToCreate, expectedPdb)

			r.Log.Debugf("Need create pdb %s", expectedPdb.Name)
		}
	}

	if len(currentPdbs) > 0 {
		diff.NeedDelete = true
		for _, pdb := range currentPdbs {
			diff.Diff.WriteString(fmt.Sprintf("Need delete pdb %s\n", pdb.Name))
		}
	}

	data["newPdbs"] = pdbToCreate
	data["pdbs"] = pdbToUpdate
	data["oldPdbs"] = currentPdbs

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *PdbReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    PdbCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *PdbReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Pdbs successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, PdbCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    PdbCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
