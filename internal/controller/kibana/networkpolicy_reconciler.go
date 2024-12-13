package kibana

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NetworkPolicyCondition shared.ConditionName = "NetworkPolicyReady"
	NetworkPolicyPhase     shared.PhaseName     = "NetworkPolicy"
)

type networkPolicyReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newNetworkPolicyReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &networkPolicyReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			NetworkPolicyPhase,
			NetworkPolicyCondition,
			recorder,
		),
	}
}

// Read existing network policy
func (r *networkPolicyReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	npList := &networkingv1.NetworkPolicyList{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current network policies
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, kibanacrd.KibanaAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, npList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read network policies")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(npList.Items))

	// Generate expected network policy
	expectedNps, err := buildNetworkPolicies(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate network policies")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedNps))

	return read, res, nil
}
