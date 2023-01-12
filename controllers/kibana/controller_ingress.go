package kibana

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IngressCondition = "IngressReady"
	IngressPhase     = "Ingress"
)

type IngressReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewIngressReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &IngressReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "ingress",
	}
}

// Name return the current phase
func (r *IngressReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *IngressReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, IngressCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   IngressCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = IngressPhase
	}

	return res, nil
}

// Read existing ingress
func (r *IngressReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	ingress := &networkingv1.Ingress{}

	// Read current ingress
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, ingress); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read ingress")
		}
		ingress = nil
	}
	data["currentIngress"] = ingress

	// Generate expected ingress
	expectedIngress, err := BuildIngress(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate ingress")
	}
	data["expectedIngress"] = expectedIngress

	return res, nil
}

// Create will create ingress
func (r *IngressReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newIngresses")
	if err != nil {
		return res, err
	}
	expectedIngresses := d.([]networkingv1.Ingress)

	for _, ingress := range expectedIngresses {
		if err = r.Client.Create(ctx, &ingress); err != nil {
			return res, errors.Wrapf(err, "Error when create ingress %s", ingress.Name)
		}
	}

	return res, nil
}

// Update will update ingress
func (r *IngressReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "ingresses")
	if err != nil {
		return res, err
	}
	expectedIngresses := d.([]networkingv1.Ingress)

	for _, ingress := range expectedIngresses {
		if err = r.Client.Update(ctx, &ingress); err != nil {
			return res, errors.Wrapf(err, "Error when update ingress %s", ingress.Name)
		}
	}

	return res, nil
}

// Delete permit to delete ingress
func (r *IngressReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldIngresses")
	if err != nil {
		return res, err
	}
	oldIngresses := d.([]networkingv1.Ingress)

	for _, ingress := range oldIngresses {
		if err = r.Client.Delete(ctx, &ingress); err != nil {
			return res, errors.Wrapf(err, "Error when delete ingress %s", ingress.Name)
		}
	}

	return res, nil
}

// Diff permit to check if ingress is up to date
func (r *IngressReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	var d any

	d, err = helper.Get(data, "currentIngress")
	if err != nil {
		return diff, res, err
	}
	currentIngress := d.(*networkingv1.Ingress)

	d, err = helper.Get(data, "expectedIngress")
	if err != nil {
		return diff, res, err
	}
	expectedIngress := d.(*networkingv1.Ingress)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	ingressToUpdate := make([]networkingv1.Ingress, 0)
	ingressToCreate := make([]networkingv1.Ingress, 0)
	ingressToDelete := make([]networkingv1.Ingress, 0)

	if currentIngress == nil {
		if expectedIngress != nil {
			// Create new ingress
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create ingress %s\n", expectedIngress.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, expectedIngress, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedIngress); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on ingress %s", expectedIngress.Name)
			}

			ingressToCreate = append(ingressToCreate, *expectedIngress)
		}
	} else {

		if expectedIngress != nil {
			// Check if need to update ingress
			patchResult, err := patch.DefaultPatchMaker.Calculate(currentIngress, expectedIngress, patch.CleanMetadata(), patch.IgnoreStatusFields())
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when diffing ingress %s", currentIngress.Name)
			}
			if !patchResult.IsEmpty() {
				updatedIngress := patchResult.Patched.(*networkingv1.Ingress)
				diff.NeedUpdate = true
				diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedIngress.Name, string(patchResult.Patch)))
				ingressToUpdate = append(ingressToUpdate, *updatedIngress)
				r.Log.Debugf("Need update ingress %s", updatedIngress.Name)
			}
		} else {
			// Need delete ingress
			diff.NeedDelete = true
			diff.Diff.WriteString(fmt.Sprintf("Delete ingress %s\n", currentIngress.Name))
			ingressToDelete = append(ingressToDelete, *currentIngress)
		}

	}

	data["newIngresses"] = ingressToCreate
	data["ingresses"] = ingressToUpdate
	data["oldIngresses"] = ingressToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *IngressReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    IngressCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *IngressReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Ingress successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, IngressCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    IngressCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
