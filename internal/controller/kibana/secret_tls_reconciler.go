package kibana

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	TlsCondition shared.ConditionName = "TlsReady"
	TlsPhase     shared.PhaseName     = "Tls"

	// AnnotationForceRenewTLS forces a full CA + leaf rotation (read as "true").
	AnnotationForceRenewTLS = "kibana.k8s.webcenter.fr/force-renew-tls"
	// AnnotationForceRenewCertificates forces leaf-only renewal (read as "true").
	AnnotationForceRenewCertificates = "kibana.k8s.webcenter.fr/force-renew-certificates"
)

// tlsReconciler wraps the TLS rotation saga for Kibana self-managed TLS.
// BYO / TLS-disabled paths are short-circuited in Read.
type tlsReconciler struct {
	workflow.WorkflowStepReconcilerActionWithDiff[*kibanacrd.Kibana, client.Object]
}

func newTlsReconciler(c client.Client, recorder record.EventRecorder, log *logrus.Entry) multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, client.Object] {
	saga := rotation.NewTLSStep[*kibanacrd.Kibana](
		c,
		TlsPhase,
		TlsCondition,
		recorder,
		common.FieldManager,
		selfmanaged.NewSelfManagedBackend[*kibanacrd.Kibana](),
		certificate.TLSSpecProviderFunc[*kibanacrd.Kibana](kibanaTLSSpec),
		rotation.WithConvergenceCheck(kibanaConvergenceCheck(c)),
		rotation.WithCertificateCustomizer(kibanaCARenewalCustomizer(c, log)),
		rotation.WithForceRegenerateAllAnnotation[*kibanacrd.Kibana](AnnotationForceRenewTLS),
		rotation.WithForceRegenerateLeafAnnotation[*kibanacrd.Kibana](AnnotationForceRenewCertificates),
		rotation.WithLabelsDecorator(func(o *kibanacrd.Kibana, obj client.Object) {
			obj.SetLabels(getLabels(o))
		}),
		rotation.WithAnnotationsDecorator(func(o *kibanacrd.Kibana, obj client.Object) {
			obj.SetAnnotations(getAnnotations(o))
			obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
		}),
	)
	return &tlsReconciler{WorkflowStepReconcilerActionWithDiff: saga}
}

// Read short-circuits the saga when TLS is disabled or user-managed (BYO);
// otherwise performs legacy-CA cleanup and delegates to the saga.
func (r *tlsReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (multiphase.MultiPhaseRead[client.Object], reconcile.Result, error) {
	if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
		return multiphase.NewMultiPhaseRead[client.Object](), reconcile.Result{}, nil
	}

	if err := cleanupLegacyKibanaCASecret(ctx, r.Client(), o, logger); err != nil {
		logger.Warnf("Failed to clean up legacy Kibana CA secret: %s", err.Error())
	}

	return r.WorkflowStepReconcilerActionWithDiff.Read(ctx, o, data, logger)
}

// kibanaTLSSpec builds the computed TLSSpec for a Kibana CR.
func kibanaTLSSpec(o *kibanacrd.Kibana) certificate.TLSSpec {
	validityDays := 365
	if o.Spec.Tls.ValidityDays != nil {
		validityDays = *o.Spec.Tls.ValidityDays
	}

	renewalDays := 30
	if o.Spec.Tls.RenewalDays != nil {
		renewalDays = *o.Spec.Tls.RenewalDays
	}

	spec := certificate.TLSSpec{
		SecretName:       GetSecretNameForTls(o),
		CommonName:       o.Name,
		CACommonName:     fmt.Sprintf("%s-api", o.Name),
		LeafValidityDays: validityDays,
		CAValidityDays:   validityDays,
		RenewalDays:      renewalDays,
		Subject: certificate.CertificateSubject{
			Organizations:       []string{o.Name},
			OrganizationalUnits: []string{"api"},
			Countries:           []string{"internal"},
			Localities:          []string{"internal"},
			Provinces:           []string{"internal"},
		},
		DNSNames: []string{
			GetServiceName(o),
			fmt.Sprintf("%s.%s", GetServiceName(o), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetServiceName(o), o.Namespace),
		},
	}

	if o.Spec.Tls.SelfSignedCertificate != nil {
		spec.DNSNames = append(spec.DNSNames, o.Spec.Tls.SelfSignedCertificate.AltNames...)
		spec.IPAddresses = append(spec.IPAddresses, o.Spec.Tls.SelfSignedCertificate.AltIps...)
	}

	// Defense-in-depth: CRD MaxItems validation should prevent this, but
	// truncate excessive SANs to avoid oversized certificates (CWE-400).
	// Truncation is silent (no logger available in TLSSpecProviderFunc).
	if len(spec.DNSNames) > 64 {
		spec.DNSNames = spec.DNSNames[:64]
	}
	if len(spec.IPAddresses) > 64 {
		spec.IPAddresses = spec.IPAddresses[:64]
	}

	applyKeyComplexity(&spec, o.Spec.Tls.KeyComplexity, o.Spec.Tls.KeySize)

	return spec
}

// applyKeyComplexity maps the CR's KeyComplexity/legacy KeySize to TLSSpec
// key algorithm/curve/size (mirror of elasticsearch.applyKeyComplexity).
func applyKeyComplexity(spec *certificate.TLSSpec, complexity string, legacyKeySize *int) {
	switch complexity {
	case "rsa-2048":
		spec.KeyAlgorithm = certificate.KeyAlgorithmRSA
		spec.KeySize = 2048
	case "rsa-4096":
		spec.KeyAlgorithm = certificate.KeyAlgorithmRSA
		spec.KeySize = 4096
	case "ecdsa-p256":
		spec.KeyAlgorithm = certificate.KeyAlgorithmECDSA
		spec.Curve = certificate.CurveP256
	case "ecdsa-p384":
		spec.KeyAlgorithm = certificate.KeyAlgorithmECDSA
		spec.Curve = certificate.CurveP384
	case "ecdsa-p521":
		spec.KeyAlgorithm = certificate.KeyAlgorithmECDSA
		spec.Curve = certificate.CurveP521
	default:
		spec.KeyAlgorithm = certificate.KeyAlgorithmRSA
		spec.KeySize = 2048
		if legacyKeySize != nil {
			spec.KeySize = *legacyKeySize
		}
	}
}

// kibanaCARenewalCustomizer forces a CA rotation when the current CA is within
// caRenewalDays of expiry (sets AnnotationForceRenewTLS on o).
func kibanaCARenewalCustomizer(c client.Client, log *logrus.Entry) certificate.CertificateCustomizer[*kibanacrd.Kibana] {
	return certificate.CertificateCustomizerFunc[*kibanacrd.Kibana](func(o *kibanacrd.Kibana, base certificate.TLSSpec) (certificate.TLSSpec, error) {
		if o.Spec.Tls.CaRenewalDays == nil {
			return base, nil
		}

		caSecret := &corev1.Secret{}
		if err := c.Get(context.Background(), types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPki(o)}, caSecret); err != nil {
			if k8serrors.IsNotFound(err) {
				return base, nil // CA not yet generated; bootstrap creates it
			}
			return base, err
		}

		crt, err := certParse(caSecret.Data["ca.crt"])
		if err != nil {
			return base, nil // unparseable CA; avoid breaking bootstrap
		}

		// Defense-in-depth: the CA secret is operator-generated and should always
		// contain a CA certificate, but if it doesn't, don't trust it for renewal.
		if !crt.IsCA {
			log.Warnf("CA certificate in secret %s/%s is not a CA; skipping CA renewal check", o.Namespace, GetSecretNameForPki(o))
			return base, nil
		}

		needRenew, err := pki.NeedRenewCertificate(crt, time.Duration(*o.Spec.Tls.CaRenewalDays)*24*time.Hour, log)
		if err != nil || !needRenew {
			return base, nil
		}

		if o.Annotations == nil {
			o.Annotations = map[string]string{}
		}
		o.Annotations[AnnotationForceRenewTLS] = "true"

		return base, nil
	})
}

// kibanaConvergenceCheck gates the Rotate→Converge transition on the Kibana
// Deployment having rolled to the latest generation.
func kibanaConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*kibanacrd.Kibana] {
	return func(ctx context.Context, o *kibanacrd.Kibana, data map[string]any) (bool, error) {
		// In envtest there is no kubelet to roll pods, so the convergence check
		// would never pass and the CA rotation saga would stall in "Converge".
		if common.IsEnvtest() {
			return true, nil
		}

		dpl := &appv1.Deployment{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetDeploymentName(o)}, dpl); err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "Error when read Kibana deployment")
		}

		if dpl.Status.ObservedGeneration < dpl.Generation {
			return false, nil
		}

		expectedReplicas := o.Spec.Deployment.Replicas
		if dpl.Status.UpdatedReplicas < expectedReplicas || dpl.Status.AvailableReplicas < expectedReplicas {
			return false, nil
		}

		return true, nil
	}
}

// cleanupLegacyKibanaCASecret deletes the pre-saga PKI secret <name>-pki-kb once
// the new saga CA secret <name>-tls-kb-ca exists. Best-effort.
func cleanupLegacyKibanaCASecret(ctx context.Context, c client.Client, o *kibanacrd.Kibana, logger *logrus.Entry) error {
	// The new CA must exist first (the saga has generated it at least once).
	newCA := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPki(o)}, newCA); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	legacyName := fmt.Sprintf("%s-pki-kb", o.Name)
	legacy := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: legacyName}, legacy); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := c.Delete(ctx, legacy); err != nil {
		return err
	}
	logger.Infof("Deleted legacy Kibana CA secret %s/%s", o.Namespace, legacyName)

	return nil
}

// certParse parses the first CERTIFICATE PEM block.
func certParse(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("no certificate found in PEM data")
	}
	return x509.ParseCertificate(block.Bytes)
}
