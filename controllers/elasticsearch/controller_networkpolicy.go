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
	NetworkPolicyCondition = "NetworkPolicyReady"
	NetworkPolicyPhase     = "NetworkPolicy"
)

type NetworkPolicyReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewNetworkPolicyReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &NetworkPolicyReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "networkPolicy",
	}
}

// Name return the current phase
func (r *NetworkPolicyReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *NetworkPolicyReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, NetworkPolicyCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   NetworkPolicyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = NetworkPolicyPhase
	}

	return res, nil
}

// Read existing network policy
func (r *NetworkPolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	np := &networkingv1.NetworkPolicy{}

	// Read current ingress
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetNetworkPolicyName(o)}, np); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read network policy")
		}
		np = nil
	}
	data["currentNetworkPolicy"] = np

	// Generate expected network policy
	expectedNp, err := BuildNetworkPolicy(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate network policy")
	}
	data["expectedNetworkPolicy"] = expectedNp

	return res, nil
}

// Create will create network policy
func (r *NetworkPolicyReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newNetworkPolicies")
	if err != nil {
		return res, err
	}
	expectedNps := d.([]networkingv1.NetworkPolicy)

	for _, np := range expectedNps {
		if err = r.Client.Create(ctx, &np); err != nil {
			return res, errors.Wrapf(err, "Error when create network policy %s", np.Name)
		}
	}

	return res, nil
}

// Update will update network policy
func (r *NetworkPolicyReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "networkPolicies")
	if err != nil {
		return res, err
	}
	expectedNps := d.([]networkingv1.NetworkPolicy)

	for _, np := range expectedNps {
		if err = r.Client.Update(ctx, &np); err != nil {
			return res, errors.Wrapf(err, "Error when update network policy %s", np.Name)
		}
	}

	return res, nil
}

// Delete permit to delete network policy
func (r *NetworkPolicyReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldNetworkPolicies")
	if err != nil {
		return res, err
	}
	oldNetworkPolicies := d.([]networkingv1.NetworkPolicy)

	for _, np := range oldNetworkPolicies {
		if err = r.Client.Delete(ctx, &np); err != nil {
			return res, errors.Wrapf(err, "Error when delete network policy %s", np.Name)
		}
	}

	return res, nil
}

// Diff permit to check if network policy is up to date
func (r *NetworkPolicyReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentNetworkPolicy")
	if err != nil {
		return diff, res, err
	}
	currentNetworkPolicy := d.(*networkingv1.NetworkPolicy)

	d, err = helper.Get(data, "expectedNetworkPolicy")
	if err != nil {
		return diff, res, err
	}
	expectedNetworkPolicy := d.(*networkingv1.NetworkPolicy)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	networkPolicyToUpdate := make([]networkingv1.NetworkPolicy, 0)
	networkPolicyToCreate := make([]networkingv1.NetworkPolicy, 0)
	networkPolicyToDelete := make([]networkingv1.NetworkPolicy, 0)

	if currentNetworkPolicy == nil {
		// Create new network policy
		diff.NeedCreate = true
		diff.Diff.WriteString(fmt.Sprintf("Create network policy %s\n", expectedNetworkPolicy.Name))

		// Set owner
		err = ctrl.SetControllerReference(o, expectedNetworkPolicy, r.Scheme)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference")
		}

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedNetworkPolicy); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on network policy %s", expectedNetworkPolicy.Name)
		}

		networkPolicyToCreate = append(networkPolicyToCreate, *expectedNetworkPolicy)
	} else {
		// Check if need to update ingress
		patchResult, err := patch.DefaultPatchMaker.Calculate(currentNetworkPolicy, expectedNetworkPolicy, patch.CleanMetadata(), patch.IgnoreStatusFields())
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when diffing network policy %s", currentNetworkPolicy.Name)
		}
		if !patchResult.IsEmpty() {
			updatedNetworkPolicy := patchResult.Patched.(*networkingv1.NetworkPolicy)
			diff.NeedUpdate = true
			diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedNetworkPolicy.Name, string(patchResult.Patch)))
			networkPolicyToUpdate = append(networkPolicyToUpdate, *updatedNetworkPolicy)
			r.Log.Debugf("Need update ingress %s", updatedNetworkPolicy.Name)
		}
	}

	data["newNetworkPolicies"] = networkPolicyToCreate
	data["networkPolicies"] = networkPolicyToUpdate
	data["oldNetworkPolicies"] = networkPolicyToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *NetworkPolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    NetworkPolicyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *NetworkPolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Network policy successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, NetworkPolicyCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    NetworkPolicyCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
