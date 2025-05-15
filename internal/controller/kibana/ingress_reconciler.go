package kibana

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	IngressCondition shared.ConditionName = "IngressReady"
	IngressPhase     shared.PhaseName     = "Ingress"
)

type ingressReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.Ingress]
}

func newIngressReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.Ingress]) {
	return &ingressReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *networkingv1.Ingress](
			client,
			IngressPhase,
			IngressCondition,
			recorder,
		),
	}
}

// Read existing ingress
func (r *ingressReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*networkingv1.Ingress], res reconcile.Result, err error) {
	ingress := &networkingv1.Ingress{}
	read = multiphase.NewMultiPhaseRead[*networkingv1.Ingress]()

	// Read current ingress
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, ingress); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read ingress")
		}
		ingress = nil
	}
	if ingress != nil {
		read.AddCurrentObject(ingress)
	}

	// Generate expected ingress
	expectedIngresses, err := buildIngresses(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate ingress")
	}
	read.SetExpectedObjects(expectedIngresses)

	return read, res, nil
}
