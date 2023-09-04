package kibana

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	o := resource.(*kibanacrd.Kibana)
	npList := &networkingv1.NetworkPolicyList{}

	// Read current network policies
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, kibanacrd.KibanaAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, npList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read network policies")
	}
	data["currentObjects"] = npList.Items

	// Generate expected network policy
	expectedNps, err := BuildNetworkPolicies(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate network policies")
	}
	data["expectedObjects"] = expectedNps

	return res, nil
}

// Diff permit to check if network policy is up to date
func (r *NetworkPolicyReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdListDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *NetworkPolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, NetworkPolicyCondition, NetworkPolicyPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *NetworkPolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, NetworkPolicyCondition, NetworkPolicyPhase)
}
