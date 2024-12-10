package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationSecretCondition shared.ConditionName = "ApplicationSecretReady"
	ApplicationSecretPhase     shared.PhaseName     = "ApplicationSecret"
)

type applicationSecretReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newApplicationSecretReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &applicationSecretReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			ApplicationSecretPhase,
			ApplicationSecretCondition,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
	}
}

// Read existing secret
func (r *applicationSecretReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	currentApplicationSecret := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForApplication(o)}, currentApplicationSecret); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForApplication(o))
		}
		currentApplicationSecret = nil
	}
	if currentApplicationSecret != nil {
		read.SetCurrentObjects([]client.Object{currentApplicationSecret})
	}

	// Generate expected secret
	expectedApplicationSecrets, err := buildApplicationSecrets(o)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForApplication(o))
	}

	// Never update existing credentials
	if currentApplicationSecret != nil {
		expectedApplicationSecrets[0].Data = currentApplicationSecret.Data
	}

	read.SetExpectedObjects(helper.ToSliceOfObject(expectedApplicationSecrets))

	return read, res, nil
}
