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
	LoadBalancerCondition = "LoadBalancerReady"
	LoadBalancerPhase     = "LoadBalancer"
)

type LoadBalancerReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewLoadBalancerReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &LoadBalancerReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "loadBalancer",
	}
}

// Name return the current phase
func (r *LoadBalancerReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *LoadBalancerReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, LoadBalancerCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   LoadBalancerCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = LoadBalancerPhase
	}

	return res, nil
}

// Read existing load balancer
func (r *LoadBalancerReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	lb := &corev1.Service{}

	// Read current load balancer
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLoadBalancerName(o)}, lb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read load balancer")
		}
		lb = nil
	}
	data["currentLoadBalancer"] = lb

	// Generate expected load balancer
	expectedLb, err := BuildLoadbalancer(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate load balancer")
	}
	data["expectedLoadBalancer"] = expectedLb

	return res, nil
}

// Create will create load balancer
func (r *LoadBalancerReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newLoadBalancers")
	if err != nil {
		return res, err
	}
	expectedLbs := d.([]corev1.Service)

	for _, lb := range expectedLbs {
		if err = r.Client.Create(ctx, &lb); err != nil {
			return res, errors.Wrapf(err, "Error when create load balancer %s", lb.Name)
		}
	}

	return res, nil
}

// Update will update load balancer
func (r *LoadBalancerReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "loadBalancers")
	if err != nil {
		return res, err
	}
	expectedLbs := d.([]corev1.Service)

	for _, lb := range expectedLbs {
		if err = r.Client.Update(ctx, &lb); err != nil {
			return res, errors.Wrapf(err, "Error when update load balancer %s", lb.Name)
		}
	}

	return res, nil
}

// Delete permit to delete load balancer
func (r *LoadBalancerReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldLoadBalancers")
	if err != nil {
		return res, err
	}
	oldLbs := d.([]corev1.Service)

	for _, lb := range oldLbs {
		if err = r.Client.Delete(ctx, &lb); err != nil {
			return res, errors.Wrapf(err, "Error when delete load balancer %s", lb.Name)
		}
	}

	return res, nil
}

// Diff permit to check if load balancer is up to date
func (r *LoadBalancerReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentLoadBalancer")
	if err != nil {
		return diff, res, err
	}
	currentLb := d.(*corev1.Service)

	d, err = helper.Get(data, "expectedLoadBalancer")
	if err != nil {
		return diff, res, err
	}
	expectedLb := d.(*corev1.Service)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	lbToUpdate := make([]corev1.Service, 0)
	lbToCreate := make([]corev1.Service, 0)
	lbToDelete := make([]corev1.Service, 0)

	if currentLb == nil {
		if expectedLb != nil {
			// Create new load balancer
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create load balancer %s\n", expectedLb.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, expectedLb, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedLb); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on load balancer %s", expectedLb.Name)
			}

			lbToCreate = append(lbToCreate, *expectedLb)
		}
	} else {
		if expectedLb != nil {
			// Check if need to update load balancer
			patchResult, err := patch.DefaultPatchMaker.Calculate(currentLb, expectedLb, patch.CleanMetadata(), patch.IgnoreStatusFields())
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when diffing load balancer %s", currentLb.Name)
			}
			if !patchResult.IsEmpty() {
				updatedLb := patchResult.Patched.(*corev1.Service)
				diff.NeedUpdate = true
				diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedLb.Name, string(patchResult.Patch)))
				lbToUpdate = append(lbToUpdate, *updatedLb)
				r.Log.Debugf("Need update load balancer %s", updatedLb.Name)
			}
		} else {
			// Need delete load balancer
			diff.NeedDelete = true
			diff.Diff.WriteString(fmt.Sprintf("Delete load balancer %s\n", currentLb.Name))
			lbToDelete = append(lbToDelete, *currentLb)
		}
	}

	data["newLoadBalancers"] = lbToCreate
	data["loadBalancers"] = lbToUpdate
	data["oldLoadBalancers"] = lbToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *LoadBalancerReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    LoadBalancerCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LoadBalancerReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Load balancer successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, LoadBalancerCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    LoadBalancerCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
