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
	CredentialCondition shared.ConditionName = "CredentialReady"
	CredentialPhase     shared.PhaseName     = "Credential"
)

type credentialReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newCredentialReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &credentialReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			CredentialPhase,
			CredentialCondition,
			recorder,
		),
	}
}

// Read existing secret
func (r *credentialReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	currentCredential := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, currentCredential); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
		}
		currentCredential = nil
	}
	if currentCredential != nil {
		read.SetCurrentObjects([]client.Object{currentCredential})
	}

	// Generate expected secret
	expectedCredentials, err := buildCredentialSecrets(o)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCredentials(o))
	}

	// Never update existing credentials
	if currentCredential != nil {
		expectedCredentials[0].Data = currentCredential.Data
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedCredentials))

	return read, res, nil
}
