package logstash

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NetworkPolicyCondition common.ConditionName = "NetworkPolicyReady"
	NetworkPolicyPhase     common.PhaseName     = "NetworkPolicy"
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
	return r.StdConfigure(ctx, req, resource, NetworkPolicyCondition, NetworkPolicyPhase)
}

// Read existing network policy
func (r *NetworkPolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)
	np := &networkingv1.NetworkPolicy{}
	filebeatList := &beatcrd.FilebeatList{}
	oList := make([]client.Object, 0)
	var oListTmp []client.Object

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
	oListTmp = helper.ToSliceOfObject(filebeatList.Items)
	for _, fb := range oListTmp {
		if fb.GetNamespace() != o.Namespace {
			oList = append(oList, fb)
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
	return r.StdOnError(ctx, resource, data, currentErr, NetworkPolicyCondition, NetworkPolicyPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *NetworkPolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, NetworkPolicyCondition, NetworkPolicyPhase)
}
