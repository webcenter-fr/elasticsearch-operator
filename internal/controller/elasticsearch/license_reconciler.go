package elasticsearch

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LicenseCondition shared.ConditionName = "LicenseReady"
	LicensePhase     shared.PhaseName     = "License"
)

type licenseReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newLicenseReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &licenseReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			LicensePhase,
			LicenseCondition,
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

// Read existing license
func (r *licenseReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	license := &elasticsearchapicrd.License{}
	s := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current license
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLicenseName(o)}, license); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read license")
		}
		license = nil
	}
	if license != nil {
		read.SetCurrentObjects([]client.Object{license})
	}

	// Check if license is expected
	if o.Spec.LicenseSecretRef != nil {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LicenseSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LicenseSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.LicenseSecretRef.Name)
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		s = nil
	}

	// Generate expected license
	expectedLicenses, err := buildLicenses(o, s)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate license")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedLicenses))

	return read, res, nil
}
