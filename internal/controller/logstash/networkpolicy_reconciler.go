package logstash

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
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
	o := resource.(*logstashcrd.Logstash)
	np := &networkingv1.NetworkPolicy{}
	read = controller.NewBasicMultiPhaseRead()
	filebeatList := &beatcrd.FilebeatList{}
	oList := make([]client.Object, 0)
	var oListTmp []client.Object

	// Read current network policy
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetNetworkPolicyName(o)}, np); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read network policy")
		}
		np = nil
	}
	if np != nil {
		read.SetCurrentObjects([]client.Object{np})
	}

	// Read filebeat referer
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.logstashRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client().List(context.Background(), filebeatList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read filebeat")
	}
	oListTmp = helper.ToSliceOfObject(filebeatList.Items)
	for _, fb := range oListTmp {
		if fb.GetNamespace() != o.Namespace {
			oList = append(oList, fb)
		}
	}

	// Generate expected network policy
	expectedNps, err := buildNetworkPolicies(o, oList)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate network policy")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedNps))

	return read, res, nil
}
