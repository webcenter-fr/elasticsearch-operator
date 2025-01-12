package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceAccountCondition shared.ConditionName = "ServiceAccountReady"
	ServiceAccountPhase     shared.PhaseName     = "ServiceAccount"
)

type serviceAccountReconciler struct {
	controller.MultiPhaseStepReconcilerAction
	isOpenshift bool
}

func newServiceAccountReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &serviceAccountReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			ServiceAccountPhase,
			ServiceAccountCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing service account
func (r *serviceAccountReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	serviceAccount := &corev1.ServiceAccount{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current service account
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetServiceAccountName(o)}, serviceAccount); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read service account")
		}
		serviceAccount = nil
	}
	if serviceAccount != nil {
		read.SetCurrentObjects([]client.Object{serviceAccount})
	}

	// Generate expected service account
	expectedServiceAccounts, err := buildServiceAccounts(o, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate service account")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedServiceAccounts))

	return read, res, nil
}
