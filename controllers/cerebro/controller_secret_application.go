package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationSecretCondition common.ConditionName = "ApplicationSecretReady"
	ApplicationSecretPhase     common.PhaseName     = "ApplicationSecret"
)

type ApplicationSecretReconciler struct {
	common.Reconciler
}

func NewApplicationSecretReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &ApplicationSecretReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "applicationSecret",
			}),
			Name:   "applicationSecret",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *ApplicationSecretReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, ApplicationSecretCondition, ApplicationSecretPhase)
}

// Read existing secret
func (r *ApplicationSecretReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	currentApplicationSecret := &corev1.Secret{}

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForApplication(o)}, currentApplicationSecret); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForApplication(o))
		}
		currentApplicationSecret = nil
	}
	data["currentObject"] = currentApplicationSecret

	// Generate expected secret
	expectedApplicationSecret, err := BuildApplicationSecret(o)
	if err != nil {
		return res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForApplication(o))
	}

	// Never update existing credentials
	if currentApplicationSecret != nil {
		expectedApplicationSecret.Data = currentApplicationSecret.Data
	}

	data["expectedObject"] = expectedApplicationSecret

	return res, nil
}

// Diff permit to check if credential secret is up to date
func (r *ApplicationSecretReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *ApplicationSecretReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, ApplicationSecretCondition, ApplicationSecretPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ApplicationSecretReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, ApplicationSecretCondition, ApplicationSecretPhase)
}
