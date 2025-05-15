package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	CredentialCondition shared.ConditionName = "CredentialReady"
	CredentialPhase     shared.PhaseName     = "Credential"
)

type credentialReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]
}

func newCredentialReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]) {
	return &credentialReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret](
			client,
			CredentialPhase,
			CredentialCondition,
			recorder,
		),
	}
}

// Read existing secret
func (r *credentialReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	currentCredential := &corev1.Secret{}
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()

	// Read current secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, currentCredential); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
		}
		currentCredential = nil
	}
	if currentCredential != nil {
		read.AddCurrentObject(currentCredential)
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
	read.SetExpectedObjects(expectedCredentials)

	return read, res, nil
}
