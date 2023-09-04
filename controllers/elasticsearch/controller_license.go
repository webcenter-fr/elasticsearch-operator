package elasticsearch

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
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
	LicenseCondition common.ConditionName = "LicenseReady"
	LicensePhase     common.PhaseName     = "License"
)

type LicenseReconciler struct {
	common.Reconciler
}

func NewLicenseReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &LicenseReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "license",
			}),
			Name:   "license",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *LicenseReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, LicenseCondition, LicensePhase)
}

// Read existing license
func (r *LicenseReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	license := &elasticsearchapicrd.License{}
	s := &corev1.Secret{}

	// Read current license
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLicenseName(o)}, license); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read license")
		}
		license = nil
	}
	data["currentObject"] = license

	// Check if license is expected
	if o.Spec.LicenseSecretRef != nil {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LicenseSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LicenseSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.LicenseSecretRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		s = nil
	}

	// Generate expected license
	expectedLicense, err := BuildLicense(o, s)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate license")
	}
	data["expectedObject"] = expectedLicense

	return res, nil
}

// Diff permit to check if license is up to date
func (r *LicenseReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *LicenseReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, LicenseCondition, LicensePhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LicenseReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, LicenseCondition, LicensePhase)
}
