package elasticsearch

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	LicenseCondition shared.ConditionName = "LicenseReady"
	LicensePhase     shared.PhaseName     = "License"
)

type licenseReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.License]
}

func newLicenseReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.License]) {
	return &licenseReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.License](
			client,
			LicensePhase,
			LicenseCondition,
			recorder,
		),
	}
}

// Read existing license
func (r *licenseReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*elasticsearchapicrd.License], res reconcile.Result, err error) {
	license := &elasticsearchapicrd.License{}
	s := &corev1.Secret{}
	read = multiphase.NewMultiPhaseRead[*elasticsearchapicrd.License]()

	// Read current license
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLicenseName(o)}, license); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read license")
		}
		license = nil
	}
	if license != nil {
		read.AddCurrentObject(license)
	}

	// Check if license is expected
	if o.Spec.LicenseSecretRef != nil {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LicenseSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LicenseSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.LicenseSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		s = nil
	}

	// Generate expected license
	expectedLicenses, err := buildLicenses(o, s)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate license")
	}
	read.SetExpectedObjects(expectedLicenses)

	return read, res, nil
}
