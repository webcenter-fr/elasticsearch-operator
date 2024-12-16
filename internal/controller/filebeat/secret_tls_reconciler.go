package filebeat

import (
	"context"
	"crypto/x509"
	"fmt"
	"regexp"
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
	"github.com/thoas/go-funk"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
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

// Read existing TLS secret
func (r *tlsReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)
	read = controller.NewBasicMultiPhaseRead()
	sCrt := &corev1.Secret{}
	sPki := &corev1.Secret{}
	var (
		rootCA     *goca.CA
		crt        *x509.Certificate
		crts       map[string]x509.Certificate
		secretName string
	)

	if o.Spec.Pki.IsEnabled() {
		// Read API PKI secret
		secretName = GetSecretNameForPki(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sPki); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sPki = nil
		}

		// Read API secret
		secretName = GetSecretNameForTls(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sCrt); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sCrt = nil
		}
	}

	// Load PKI
	if sPki != nil {
		// Load root CA
		rootCA, err = pki.LoadRootCA(sPki.Data["ca.key"], sPki.Data["ca.pub"], sPki.Data["ca.crt"], sPki.Data["ca.crl"], logger)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when load PKI")
		}
	}

	// Load certificates
	// Exclude ca.crt and keep only .crt entries
	if sCrt != nil {
		r := regexp.MustCompile(`^(.*)\.crt$`)
		crts = make(map[string]x509.Certificate)
		for name, data := range sCrt.Data {
			if name != "ca.crt" {
				rRes := r.FindStringSubmatch(name)
				if len(rRes) > 1 {
					crt, err = cert.LoadCertFromPem(data)
					if err != nil {
						return read, res, errors.Wrapf(err, "Error when load certificate")
					}
					crts[rRes[1]] = *crt
				}
			}
		}

	}

	data["rootCA"] = rootCA
	data["certificates"] = crts
	data["tlsSecret"] = sCrt
	data["pkiSecret"] = sPki

	return read, res, nil
}

// Diff permit to check if TLS secrets are up to date
func (r *tlsReconciler) Diff(ctx context.Context, resource object.MultiPhaseObject, read controller.MultiPhaseRead, data map[string]any, logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff controller.MultiPhaseDiff, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)
	var (
		d         any
		needRenew bool
		isUpdated bool
	)

	defaultRenewCertificate := DefaultRenewCertificate
	if o.Spec.Pki.RenewalDays != nil {
		defaultRenewCertificate = time.Duration(*o.Spec.Pki.RenewalDays) * 24 * time.Hour
	}

	d, err = helper.Get(data, "rootCA")
	if err != nil {
		return diff, res, err
	}
	rootCA := d.(*goca.CA)

	d, err = helper.Get(data, "certificates")
	if err != nil {
		return diff, res, err
	}
	crts := d.(map[string]x509.Certificate)

	d, err = helper.Get(data, "pkiSecret")
	if err != nil {
		return diff, res, err
	}
	sPki := d.(*corev1.Secret)

	d, err = helper.Get(data, "tlsSecret")
	if err != nil {
		return diff, res, err
	}
	sCrt := d.(*corev1.Secret)

	diff = controller.NewBasicMultiPhaseDiff()
	secretToUpdate := make([]client.Object, 0)
	secretToCreate := make([]client.Object, 0)
	secretToDelete := make([]client.Object, 0)

	// Generate all certificates
	if sCrt == nil || sPki == nil {
		logger.Debugf("Generate all certificates")
		diff.AddDiff("Generate new certificates")

		// Handle API certificates
		if o.Spec.Pki.IsEnabled() {

			// Generate API PKI
			tmpPki, rootCA, err := buildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate PKI")
			}
			sPki, isUpdated = updateSecret(sPki, tmpPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sPki)
			} else {
				secretToCreate = append(secretToCreate, sPki)
			}

			// Generate certificates
			tmpCrt, err := buildTlsSecret(o, rootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate certificates")
			}
			sCrt, isUpdated = updateSecret(sCrt, tmpCrt)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sCrt); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sCrt.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sCrt)
			} else {
				secretToCreate = append(secretToCreate, sCrt)
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
	if o.Spec.Pki.IsEnabled() {
		if rootCA != nil {
			certificates["rootCA"] = *rootCA.GoCertificate()
		} else {
			isRenew = true
		}
		if crts != nil {
			for name, crt := range crts {
				certificates[fmt.Sprintf("%s.crt", name)] = crt
			}
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

		if o.Spec.Pki.IsEnabled() {
			tmpPki, rootCA, err := buildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew Pki")
			}
			diff.AddDiff("Renew API Pki")
			sPki, isUpdated = updateSecret(sPki, tmpPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sPki)
			} else {
				secretToCreate = append(secretToCreate, sPki)
			}

			tmpCrt, err := buildTlsSecret(o, rootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew certificates")
			}
			diff.AddDiff("Renew API certificates")
			sCrt, isUpdated = updateSecret(sCrt, tmpCrt)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sCrt); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sCrt.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sCrt)
			} else {
				secretToCreate = append(secretToCreate, sCrt)
			}
		}

		diff.SetObjectsToCreate(secretToCreate)
		diff.SetObjectsToUpdate(secretToUpdate)
		diff.SetObjectsToDelete(secretToDelete)

		return diff, res, nil
	}

	// Check if need to add or remove certificates
	if o.Spec.Pki.IsEnabled() {
		expectedCrts := make([]string, 0, len(o.Spec.Pki.Tls))
		for key := range o.Spec.Pki.Tls {
			expectedCrts = append(expectedCrts, key)
		}
		addedCrt, deletedCrt := funk.DifferenceString(expectedCrts, funk.Keys(crts).([]string))
		for _, name := range addedCrt {
			tlsSpec := o.Spec.Pki.Tls[name]
			crt, err := generateCertificate(o, rootCA, name, &tlsSpec)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when generate certificate for %s", name)
			}
			sCrt.Data[fmt.Sprintf("%s.crt", name)] = []byte(crt.Certificate)
			sCrt.Data[fmt.Sprintf("%s.key", name)] = []byte(crt.RsaPrivateKey)
		}

		for _, name := range deletedCrt {

			// Remove entry on secret
			delete(sCrt.Data, fmt.Sprintf("%s.crt", name))
			delete(sCrt.Data, fmt.Sprintf("%s.key", name))
		}

		if len(addedCrt) > 0 || len(deletedCrt) > 0 {
			sCrt.Labels = getLabelsForTlsSecret(o)
			// Keep existing sequence to not rolling restart all nodes
			sCrt.Annotations = getAnnotations(o)

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sCrt); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sCrt.Name)
			}
			secretToUpdate = append(secretToUpdate, sCrt)
		}
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []*corev1.Secret{}
	if o.Spec.Pki.IsEnabled() {
		secrets = append(secrets, sPki, sCrt)
	}
	for _, s := range secrets {
		isUpdated := false
		if s.Name == GetSecretNameForTls(o) {
			if strDiff := localhelper.DiffLabels(getLabelsForTlsSecret(o), s.Labels); strDiff != "" {
				diff.AddDiff(strDiff)
				s.Labels = getLabelsForTlsSecret(o)
				isUpdated = true
			}
		} else if strDiff := localhelper.DiffLabels(getLabels(o), s.Labels); strDiff != "" {
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
