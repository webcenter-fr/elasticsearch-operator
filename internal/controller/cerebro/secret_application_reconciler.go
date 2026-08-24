package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ApplicationSecretCondition shared.ConditionName = "ApplicationSecretReady"
	ApplicationSecretPhase     shared.PhaseName     = "ApplicationSecret"
)

type applicationSecretReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Secret]
}

func newApplicationSecretReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Secret]) {
	return &applicationSecretReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Secret](
			client,
			ApplicationSecretPhase,
			ApplicationSecretCondition,
			recorder,
			common.FieldManager,
		),
	}
}

// Read existing secret
func (r *applicationSecretReconciler) Read(ctx context.Context, o *cerebrocrd.Cerebro, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	currentApplicationSecret := &corev1.Secret{}
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()

	// Read current secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForApplication(o)}, currentApplicationSecret); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForApplication(o))
		}
		currentApplicationSecret = nil
	}
	if currentApplicationSecret != nil {
		read.AddCurrentObject(currentApplicationSecret)
	}

	// Generate expected secret
	expectedApplicationSecrets, err := buildApplicationSecrets(o)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForApplication(o))
	}

	// Never regenerate existing credentials.
	// Legacy secrets carry the value under the `application` key: re-key it to
	// `session-key` in place so that already issued session cookies stay valid.
	if currentApplicationSecret != nil {
		if v, ok := currentApplicationSecret.Data[applicationSecretKey]; ok {
			expectedApplicationSecrets[0].Data[applicationSecretKey] = v
		} else if v, ok := currentApplicationSecret.Data["application"]; ok {
			logger.Infof("Migrate secret %s from key `application` to key `%s`", GetSecretNameForApplication(o), applicationSecretKey)
			expectedApplicationSecrets[0].Data[applicationSecretKey] = v

			// The multiphase framework applies children with Server-Side Apply, which
			// only prunes fields owned by the applying field manager. A legacy
			// `application` key owned by a different (pre-SSA) manager would therefore
			// linger forever. Explicitly remove it from the live object so the secret
			// converges to a single key.
			patch := []byte(`{"data":{"application":null}}`)
			if err = r.Client().Patch(ctx, currentApplicationSecret, client.RawPatch(types.MergePatchType, patch)); err != nil {
				return read, res, errors.Wrapf(err, "Error when remove legacy key `application` from secret %s", GetSecretNameForApplication(o))
			}
		}
	}

	common.InjectTypeMeta(r.Client().Scheme(), expectedApplicationSecrets...)
	read.SetExpectedObjects(expectedApplicationSecrets)

	return read, res, nil
}
