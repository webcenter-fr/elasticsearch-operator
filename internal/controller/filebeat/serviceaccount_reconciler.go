package filebeat

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ServiceAccountCondition shared.ConditionName = "ServiceAccountReady"
	ServiceAccountPhase     shared.PhaseName     = "ServiceAccount"
)

type serviceAccountReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ServiceAccount]
	isOpenshift bool
}

func newServiceAccountReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ServiceAccount]) {
	return &serviceAccountReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ServiceAccount](
			client,
			ServiceAccountPhase,
			ServiceAccountCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing service account
func (r *serviceAccountReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.ServiceAccount], res reconcile.Result, err error) {
	serviceAccount := &corev1.ServiceAccount{}
	read = multiphase.NewMultiPhaseRead[*corev1.ServiceAccount]()

	// Read current service account
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetServiceAccountName(o)}, serviceAccount); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read service account")
		}
		serviceAccount = nil
	}
	if serviceAccount != nil {
		read.AddCurrentObject(serviceAccount)
	}

	// Generate expected service account
	expectedServiceAccounts, err := buildServiceAccounts(o, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate service account")
	}
	read.SetExpectedObjects(expectedServiceAccounts)

	return read, res, nil
}
