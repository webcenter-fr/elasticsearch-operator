package kibana

import (
	"context"
	"crypto/x509"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/goca"
	"github.com/disaster37/goca/cert"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TlsCondition            shared.ConditionName = "TlsReady"
	TlsPhase                shared.PhaseName     = "Tls"
	DefaultRenewCertificate                      = -time.Hour * 24 * 30 // 30 days before expired
)

type tlsReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newTlsReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &tlsReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			TlsPhase,
			TlsCondition,
			recorder,
		),
	}
}

// Read existing transport TLS secret
func (r *tlsReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	read = controller.NewBasicMultiPhaseRead()
	sApi := &corev1.Secret{}
	sApiPki := &corev1.Secret{}
	var (
		apiRootCA  *goca.CA
		apiCrt     *x509.Certificate
		secretName string
	)

	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		// Read API PKI secret
		secretName = GetSecretNameForPki(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApiPki); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApiPki = nil
		}

		// Read API secret
		secretName = GetSecretNameForTls(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApi); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApi = nil
		}
	}

	// Load API PKI
	if sApiPki != nil {
		// Load root CA
		apiRootCA, err = pki.LoadRootCA(sApiPki.Data["ca.key"], sApiPki.Data["ca.pub"], sApiPki.Data["ca.crt"], sApiPki.Data["ca.crl"], logger)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when load PKI")
		}
	}

	// Load API certificate
	if sApi != nil {
		apiCrt, err = cert.LoadCertFromPem(sApi.Data["tls.crt"])
		if err != nil {
			return read, res, errors.Wrapf(err, "Error when load certificate")
		}
	}

	data["apiRootCA"] = apiRootCA
	data["apiCertificate"] = apiCrt
	data["apiTlsSecret"] = sApi
	data["apiPkiSecret"] = sApiPki

	return read, res, nil
}

// Diff permit to check if transport secrets are up to date
func (r *tlsReconciler) Diff(ctx context.Context, resource object.MultiPhaseObject, read controller.MultiPhaseRead, data map[string]any, logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff controller.MultiPhaseDiff, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	var (
		d         any
		needRenew bool
		isUpdated bool
	)

	defaultRenewCertificate := DefaultRenewCertificate
	if o.Spec.Tls.RenewalDays != nil {
		defaultRenewCertificate = time.Duration(*o.Spec.Tls.RenewalDays) * 24 * time.Hour
	}

	d, err = helper.Get(data, "apiRootCA")
	if err != nil {
		return diff, res, err
	}
	apiRootCA := d.(*goca.CA)

	d, err = helper.Get(data, "apiCertificate")
	if err != nil {
		return diff, res, err
	}
	apiCrt := d.(*x509.Certificate)

	d, err = helper.Get(data, "apiPkiSecret")
	if err != nil {
		return diff, res, err
	}
	sApiPki := d.(*corev1.Secret)

	d, err = helper.Get(data, "apiTlsSecret")
	if err != nil {
		return diff, res, err
	}
	sApi := d.(*corev1.Secret)

	diff = controller.NewBasicMultiPhaseDiff()
	secretToUpdate := make([]client.Object, 0)
	secretToCreate := make([]client.Object, 0)
	secretToDelete := make([]client.Object, 0)

	// Generate all certificates
	if sApi == nil || sApiPki == nil {
		logger.Debugf("Generate all certificates")
		diff.AddDiff("Generate new certificates")

		// Handle API certificates
		if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {

			// Generate API PKI
			tmpApiPki, apiRootCA, err := buildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate PKI")
			}
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				secretToCreate = append(secretToCreate, sApiPki)
			}

			// Generate API certificate
			tmpApi, err := buildTlsSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate certificate")
			}
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApi)
			} else {
				secretToCreate = append(secretToCreate, sApi)
			}
		}

		diff.SetObjectsToCreate(secretToCreate)
		diff.SetObjectsToUpdate(secretToUpdate)
		diff.SetObjectsToDelete(secretToDelete)

		return diff, res, nil
	}

	// Check if certificates will expire
	isRenew := false
	certificates := map[string]x509.Certificate{}
	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		if apiRootCA != nil {
			certificates["apiPki"] = *apiRootCA.GoCertificate()
		} else {
			isRenew = true
		}
		if apiCrt != nil {
			certificates["apiCrt"] = *apiCrt
		} else {
			isRenew = true
		}
	}

	if !isRenew {
		// Check certificate validity if all certificates exists
		for name, crt := range certificates {
			needRenew, err = pki.NeedRenewCertificate(&crt, defaultRenewCertificate, logger)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when check expiration of %s certificate", name)
			}
			if needRenew {
				isRenew = true
				break
			}
		}
	}

	if isRenew {
		logger.Debugf("Renew all certificates")

		if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
			tmpApiPki, apiRootCA, err := buildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew Pki")
			}
			diff.AddDiff("Renew API Pki")
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				secretToCreate = append(secretToCreate, sApiPki)
			}

			tmpApi, err := buildTlsSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew certificate")
			}
			diff.AddDiff("Renew API certificate")
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApi)
			} else {
				secretToCreate = append(secretToCreate, sApi)
			}
		}

		diff.SetObjectsToCreate(secretToCreate)
		diff.SetObjectsToUpdate(secretToUpdate)
		diff.SetObjectsToDelete(secretToDelete)

		return diff, res, nil
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []*corev1.Secret{}
	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		secrets = append(secrets, sApiPki, sApi)
	}
	for _, s := range secrets {
		isUpdated := false
		if strDiff := localhelper.DiffLabels(getLabels(o), s.Labels); strDiff != "" {
			diff.AddDiff(strDiff)
			s.Labels = getLabels(o)
			isUpdated = true
		}
		if strDiff := localhelper.DiffAnnotations(getAnnotations(o), s.Annotations); strDiff != "" {
			diff.AddDiff(strDiff)
			s.Annotations = getAnnotations(o)
			isUpdated = true
		}

		if isUpdated {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(s); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", s.Name)
			}
			secretToUpdate = append(secretToUpdate, s)
		}
	}

	diff.SetObjectsToCreate(secretToCreate)
	diff.SetObjectsToUpdate(secretToUpdate)
	diff.SetObjectsToDelete(secretToDelete)

	return diff, res, nil
}
