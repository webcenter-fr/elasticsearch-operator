package logstash

import (
	"context"
	"fmt"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NetworkPolicyCondition = "NetworkPolicyReady"
	NetworkPolicyPhase     = "NetworkPolicy"
)

type NetworkPolicyReconciler struct {
	common.Reconciler
}

func NewNetworkPolicyReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &NetworkPolicyReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "networkPolicy",
			}),
			Name:   "networkPolicy",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *NetworkPolicyReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, NetworkPolicyCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   NetworkPolicyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = NetworkPolicyPhase

	return res, nil
}

// Read existing network policy
func (r *NetworkPolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)
	np := &networkingv1.NetworkPolicy{}
	filebeatList := &beatcrd.FilebeatList{}
	oList := make([]client.Object, 0)

	// Read current network policy
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetNetworkPolicyName(o)}, np); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read network policy")
		}
		np = nil
	}
	data["currentObject"] = np

	// Read filebeat referer
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.logstashRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client.List(context.Background(), filebeatList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrapf(err, "Error when read filebeat")
	}
	for _, fb := range filebeatList.Items {
		if fb.Namespace != o.Namespace {
			oList = append(oList, &fb)
		}
	}

	// Generate expected network policy
	expectedNp, err := BuildNetworkPolicy(o, oList)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate network policy")
	}
	data["expectedObject"] = expectedNp

	return res, nil
}

// Diff permit to check if network policy is up to date
func (r *NetworkPolicyReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *NetworkPolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)

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
	o := resource.(*logstashcrd.Logstash)

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
