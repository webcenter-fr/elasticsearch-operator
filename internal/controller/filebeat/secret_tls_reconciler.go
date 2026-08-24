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
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	TlsCondition            shared.ConditionName = "TlsReady"
	TlsPhase                shared.PhaseName     = "Tls"
	DefaultRenewCertificate                      = -time.Hour * 24 * 30 // 30 days before expired
)

type tlsReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret]
}

func newTlsReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret]) {
	return &tlsReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.Secret](
			client,
			TlsPhase,
			TlsCondition,
			recorder,
			common.FieldManager,
		),
	}
}

// Read existing TLS secret
func (r *tlsReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
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
func (r *tlsReconciler) Diff(ctx context.Context, o *beatcrd.Filebeat, read multiphase.MultiPhaseRead[*corev1.Secret], data map[string]any, logger *logrus.Entry) (diff multiphase.MultiPhaseDiff[*corev1.Secret], res reconcile.Result, err error) {
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

	diff = multiphase.NewMultiPhaseDiff[*corev1.Secret]()

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
			sPki, isUpdated, err = updateSecret(o, sPki, tmpPki, r.Client().Scheme())
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when update secret of PKI")
			}
			if isUpdated {
				diff.AddObjectToUpdate(sPki)
			} else {
				diff.AddObjectToCreate(sPki)
			}

			// Generate certificates
			tmpCrt, err := buildTlsSecret(o, rootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate certificates")
			}
			sCrt, isUpdated, err = updateSecret(o, sCrt, tmpCrt, r.Client().Scheme())
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when update secret of certificates")
			}
			if isUpdated {
				diff.AddObjectToUpdate(sCrt)
			} else {
				diff.AddObjectToCreate(sCrt)
			}
		}

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
			sPki, isUpdated, err = updateSecret(o, sPki, tmpPki, r.Client().Scheme())
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when update secret of PKI")
			}
			if isUpdated {
				diff.AddObjectToUpdate(sPki)
			} else {
				diff.AddObjectToCreate(sPki)
			}

			tmpCrt, err := buildTlsSecret(o, rootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew certificates")
			}
			diff.AddDiff("Renew API certificates")
			sCrt, isUpdated, err = updateSecret(o, sCrt, tmpCrt, r.Client().Scheme())
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when update secret of certificates")
			}
			if isUpdated {
				diff.AddObjectToUpdate(sCrt)
			} else {
				diff.AddObjectToCreate(sCrt)
			}
		}

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

			sCrt.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
			diff.AddObjectToUpdate(sCrt)
		}
	}

	// Detect label/annotation drift so a label-only CR update re-applies the
	// TLS secrets (SSA no-op when metadata is already correct).
	if o.Spec.Pki.IsEnabled() {
		if expected := common.DriftSecretForMetadata(sPki, getLabels(o), getAnnotations(o)); expected != nil {
			diff.AddDiff("Update PKI secret metadata")
			diff.AddObjectToUpdate(expected)
		}
		if expected := common.DriftSecretForMetadata(sCrt, getLabelsForTlsSecret(o), getAnnotations(o)); expected != nil {
			diff.AddDiff("Update TLS secret metadata")
			diff.AddObjectToUpdate(expected)
		}
	}

	return diff, res, nil
}
