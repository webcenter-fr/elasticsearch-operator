package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ServiceCondition shared.ConditionName = "ServiceReady"
	ServicePhase     shared.PhaseName     = "Service"
)

type serviceReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service]
}

func newServiceReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service]) {
	return &serviceReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service](
			client,
			ServicePhase,
			ServiceCondition,
			recorder,
		),
	}
}

// Read existing services
func (r *serviceReconciler) Read(ctx context.Context, o *cerebrocrd.Cerebro, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Service], res reconcile.Result, err error) {
	service := &corev1.Service{}
	read = multiphase.NewMultiPhaseRead[*corev1.Service]()

	// Read current service
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetServiceName(o)}, service); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read service")
		}
		service = nil
	}
	if service != nil {
		read.AddCurrentObject(service)
	}

	// Generate expected service
	expectedServices, err := buildServices(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate service")
	}
	read.SetExpectedObjects(expectedServices)

	return read, res, nil
}
