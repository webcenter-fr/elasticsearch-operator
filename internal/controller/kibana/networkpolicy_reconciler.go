package kibana

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	NetworkPolicyCondition shared.ConditionName = "NetworkPolicyReady"
	NetworkPolicyPhase     shared.PhaseName     = "NetworkPolicy"
)

type networkPolicyReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.NetworkPolicy]
}

func newNetworkPolicyReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.NetworkPolicy]) {
	return &networkPolicyReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.NetworkPolicy](
			client,
			NetworkPolicyPhase,
			NetworkPolicyCondition,
			recorder,
		),
	}
}

// Read existing network policy
func (r *networkPolicyReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*networkingv1.NetworkPolicy], res reconcile.Result, err error) {
	npList := &networkingv1.NetworkPolicyList{}
	read = multiphase.NewMultiPhaseRead[*networkingv1.NetworkPolicy]()

	// Read current network policies
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, kibanacrd.KibanaAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, npList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read network policies")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(npList.Items))

	// Generate expected network policy
	expectedNps, err := buildNetworkPolicies(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate network policies")
	}
	read.SetExpectedObjects(expectedNps)

	return read, res, nil
}
