package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
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
	LicenseCondition = "LicenseReady"
	LicensePhase     = "License"
)

type LicenseReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewLicenseReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &LicenseReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "license",
	}
}

// Name return the current phase
func (r *LicenseReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *LicenseReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, LicenseCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   LicenseCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = LicensePhase
	}

	return res, nil
}

// Read existing license
func (r *LicenseReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	license := &elasticsearchapicrd.License{}
	s := &corev1.Secret{}

	// Read current license
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLicenseName(o)}, license); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read license")
		}
		license = nil
	}
	data["currentLicense"] = license

	// Check if license is expected
	if o.Spec.LicenseSecretRef != nil {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LicenseSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LicenseSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.LicenseSecretRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		s = nil
	}

	// Generate expected license
	expectedLicense, err := BuildLicense(o, s)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate license")
	}
	data["expectedLicense"] = expectedLicense

	return res, nil
}

// Create will create license
func (r *LicenseReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newLicenses")
	if err != nil {
		return res, err
	}
	expectedLicenses := d.([]elasticsearchapicrd.License)

	for _, license := range expectedLicenses {
		if err = r.Client.Create(ctx, &license); err != nil {
			return res, errors.Wrapf(err, "Error when create license %s", license.Name)
		}
	}

	return res, nil
}

// Update will update licenses
func (r *LicenseReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "licenses")
	if err != nil {
		return res, err
	}
	expectedLicenses := d.([]elasticsearchapicrd.License)

	for _, license := range expectedLicenses {
		if err = r.Client.Update(ctx, &license); err != nil {
			return res, errors.Wrapf(err, "Error when update license %s", license.Name)
		}
	}

	return res, nil
}

// Delete permit to delete licenses
func (r *LicenseReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldLicenses")
	if err != nil {
		return res, err
	}
	oldLicenses := d.([]elasticsearchapicrd.License)

	for _, license := range oldLicenses {
		if err = r.Client.Delete(ctx, &license); err != nil {
			return res, errors.Wrapf(err, "Error when delete license %s", license.Name)
		}
	}

	return res, nil
}

// Diff permit to check if license is up to date
func (r *LicenseReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentLicense")
	if err != nil {
		return diff, res, err
	}
	currentLicense := d.(*elasticsearchapicrd.License)

	d, err = helper.Get(data, "expectedLicense")
	if err != nil {
		return diff, res, err
	}
	expectedLicense := d.(*elasticsearchapicrd.License)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	licenseToUpdate := make([]elasticsearchapicrd.License, 0)
	licenseToCreate := make([]elasticsearchapicrd.License, 0)
	licenseToDelete := make([]elasticsearchapicrd.License, 0)

	if currentLicense == nil {
		if expectedLicense != nil {
			// Create new license
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create license %s\n", expectedLicense.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, expectedLicense, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedLicense); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on license %s", expectedLicense.Name)
			}

			licenseToCreate = append(licenseToCreate, *expectedLicense)
		}
	} else {

		if expectedLicense != nil {
			// Check if need to update license
			patchResult, err := patch.DefaultPatchMaker.Calculate(currentLicense, expectedLicense, patch.CleanMetadata(), patch.IgnoreStatusFields())
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when diffing license %s", currentLicense.Name)
			}
			if !patchResult.IsEmpty() {
				updatedLicense := patchResult.Patched.(*elasticsearchapicrd.License)
				diff.NeedUpdate = true
				diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedLicense.Name, string(patchResult.Patch)))
				licenseToUpdate = append(licenseToUpdate, *updatedLicense)
				r.Log.Debugf("Need update ingress %s", updatedLicense.Name)
			}
		} else {
			// Need delete license
			diff.NeedDelete = true
			diff.Diff.WriteString(fmt.Sprintf("Delete license %s\n", currentLicense.Name))
			licenseToDelete = append(licenseToDelete, *currentLicense)
		}

	}

	data["newLicenses"] = licenseToCreate
	data["licenses"] = licenseToUpdate
	data["oldLicenses"] = licenseToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *LicenseReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    LicenseCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LicenseReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "License successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, LicenseCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    LicenseCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
