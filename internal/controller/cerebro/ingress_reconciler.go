package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IngressCondition shared.ConditionName = "IngressReady"
	IngressPhase     shared.PhaseName     = "Ingress"
)

type ingressReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newIngressReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &ingressReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			IngressPhase,
			IngressCondition,
			recorder,
		),
	}
}

// Read existing ingress
func (r *ingressReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	ingress := &networkingv1.Ingress{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current ingress
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, ingress); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read ingress")
		}
		ingress = nil
	}
	if ingress != nil {
		read.SetCurrentObjects([]client.Object{ingress})
	}

	// Generate expected ingress
	expectedIngresses, err := buildIngresses(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate ingress")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedIngresses))

	return read, res, nil
}
