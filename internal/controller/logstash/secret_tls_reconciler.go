package logstash

import (
	"context"
	"fmt"
	"net"
	"sort"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	apworkflow "github.com/disaster37/operator-sdk-extra/v3/pkg/apis/workflow"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged/pernode"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
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
	AnnotationForceRenewTLS = "logstash.k8s.webcenter.fr/force-renew-tls"
	// AnnotationForceRenewCertificates forces leaf-only renewal (read as "true").
	AnnotationForceRenewCertificates = "logstash.k8s.webcenter.fr/force-renew-certificates"
)

// tlsReconciler wraps the TLS rotation saga for Logstash self-managed PKI.
// PKI-disabled / empty-tls paths are short-circuited in Read.
type tlsReconciler struct {
	workflow.WorkflowStepReconcilerActionWithDiff[*logstashcrd.Logstash, client.Object]
}

func newTlsReconciler(c client.Client, recorder record.EventRecorder) multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, client.Object] {
	saga := rotation.NewTLSStep[*logstashcrd.Logstash](
		c,
		TlsPhase,
		TlsCondition,
		recorder,
		common.FieldManager,
		pernode.NewPerNodeBackend[*logstashcrd.Logstash](&logstashNodeSpecProvider{}),
		certificate.TLSSpecProviderFunc[*logstashcrd.Logstash](logstashTLSSpec),
		rotation.WithConvergenceCheck(logstashConvergenceCheck(c)),
		rotation.WithForceRegenerateAllAnnotation[*logstashcrd.Logstash](AnnotationForceRenewTLS),
		rotation.WithForceRegenerateLeafAnnotation[*logstashcrd.Logstash](AnnotationForceRenewCertificates),
		rotation.WithLabelsDecorator(func(o *logstashcrd.Logstash, obj client.Object) {
			obj.SetLabels(getLabels(o))
		}),
		rotation.WithAnnotationsDecorator(func(o *logstashcrd.Logstash, obj client.Object) {
			obj.SetAnnotations(getAnnotations(o))
			obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
		}),
	)
	return &tlsReconciler{WorkflowStepReconcilerActionWithDiff: saga}
}

// Read short-circuits the saga when PKI is disabled or has no TLS entries;
// otherwise performs legacy-CA cleanup and delegates to the saga.
func (r *tlsReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (multiphase.MultiPhaseRead[client.Object], reconcile.Result, error) {
	if !o.Spec.Pki.IsEnabled() || len(o.Spec.Pki.Tls) == 0 {
		return multiphase.NewMultiPhaseRead[client.Object](), reconcile.Result{}, nil
	}

	if err := cleanupLegacyLogstashCASecret(ctx, r.Client(), o, logger); err != nil {
		logger.Warnf("Failed to clean up legacy Logstash CA secret: %s", err.Error())
	}

	read, res, err := r.WorkflowStepReconcilerActionWithDiff.Read(ctx, o, data, logger)
	if err != nil {
		return read, res, err
	}

	// The rotation saga does not re-apply labels/annotations on a label-only
	// CR update (its "" steady state registers current==expected). Detect and
	// fix that drift here so the saga secrets carry the operator labels after
	// an update. Only the steady state is patched: during a rotation cycle the
	// saga emits freshly decorated objects and appending stale-Data copies
	// would overwrite the new certificates.
	startPhase, _ := data["rotationStartPhase"].(apworkflow.WorkflowPhase)
	renewed, _ := data["rotationRenewed"].(bool)
	_, hasSignals := data["tls."+TlsPhase.String()].(*certificate.LayerSignals)
	if startPhase == "" && !renewed && !hasSignals {
		drift, err := buildTlsMetadataDrift(ctx, r.Client(), o)
		if err != nil {
			return read, res, err
		}
		for _, d := range drift {
			read.AddExpectedObject(d)
		}
	}

	return read, res, nil
}

// buildTlsMetadataDrift returns copies of the saga-managed CA and leaf secrets
// with corrected labels/annotations when they differ from the operator defaults
// (empty otherwise). The saga's steady state registers current==expected, so a
// label-only CR update would otherwise never reach the Secrets.
func buildTlsMetadataDrift(ctx context.Context, c client.Client, o *logstashcrd.Logstash) ([]client.Object, error) {
	out := make([]client.Object, 0, 2)
	for _, name := range []string{GetSecretNameForPki(o), GetSecretNameForTls(o)} {
		s := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: name}, s); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			}
			return nil, errors.Wrapf(err, "Error when read existing secret %s", name)
		}
		if drifted := common.DriftSecretForMetadata(s, getLabels(o), getAnnotations(o)); drifted != nil {
			out = append(out, drifted)
		}
	}
	return out, nil
}

// logstashNodeSpecProvider implements pernode.NodeSpecProvider[*logstashcrd.Logstash];
// "nodes" are the keys of spec.pki.tls.
type logstashNodeSpecProvider struct{}

func (p *logstashNodeSpecProvider) ExpectedNodeNames(o *logstashcrd.Logstash) ([]string, error) {
	names := make([]string, 0, len(o.Spec.Pki.Tls))
	for name := range o.Spec.Pki.Tls {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (p *logstashNodeSpecProvider) NodeCertSpec(o *logstashcrd.Logstash, nodeName string) (cn string, dnsNames []string, ips []string, err error) {
	tlsSpec, ok := o.Spec.Pki.Tls[nodeName]
	if !ok {
		return "", nil, nil, errors.Errorf("unknown TLS entry %q", nodeName)
	}

	// Logstash old generateCertificate only includes per-service DNS entries
	// (no global service names)
	dnsNames = make([]string, 0, (len(o.Spec.Services)*3)+len(o.Spec.Ingresses))
	for _, service := range o.Spec.Services {
		dnsNames = append(dnsNames,
			GetServiceName(o, service.Name),
			fmt.Sprintf("%s.%s", GetServiceName(o, service.Name), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetServiceName(o, service.Name), o.Namespace),
		)
	}

	for _, ingress := range o.Spec.Ingresses {
		for _, endpoint := range ingress.Spec.Rules {
			dnsNames = append(dnsNames, endpoint.Host)
		}
	}

	dnsNames = append(dnsNames, tlsSpec.AltNames...)

	ips = make([]string, 0, len(tlsSpec.AltIps))
	for _, ipStr := range tlsSpec.AltIps {
		if net.ParseIP(ipStr) == nil {
			return "", nil, nil, errors.Errorf("IP %s is not valid", ipStr)
		}
		ips = append(ips, ipStr)
	}

	return nodeName, dnsNames, ips, nil
}

// logstashTLSSpec builds the shared base TLSSpec (subject, validity, renewal,
// RSA key size). Per-node CN/DNS/IPs are supplied by NodeCertSpec.
func logstashTLSSpec(o *logstashcrd.Logstash) certificate.TLSSpec {
	validityDays := 365
	if o.Spec.Pki.ValidityDays != nil {
		validityDays = *o.Spec.Pki.ValidityDays
	}

	renewalDays := 30
	if o.Spec.Pki.RenewalDays != nil {
		renewalDays = *o.Spec.Pki.RenewalDays
	}

	keySize := 2048
	if o.Spec.Pki.KeySize != nil {
		keySize = *o.Spec.Pki.KeySize
	}

	return certificate.TLSSpec{
		SecretName:       GetSecretNameForTls(o),
		CommonName:       o.Name,
		CACommonName:     fmt.Sprintf("%s-logstash", o.Name),
		LeafValidityDays: validityDays,
		CAValidityDays:   validityDays,
		RenewalDays:      renewalDays,
		KeyAlgorithm:     certificate.KeyAlgorithmRSA,
		KeySize:          keySize,
		Subject: certificate.CertificateSubject{
			Organizations:       []string{o.Name},
			OrganizationalUnits: []string{"logstash"},
			Countries:           []string{"internal"},
			Localities:          []string{"internal"},
			Provinces:           []string{"internal"},
		},
	}
}

// logstashConvergenceCheck gates Rotate→Converge on the Logstash StatefulSet
// having rolled to the latest generation.
func logstashConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*logstashcrd.Logstash] {
	return func(ctx context.Context, o *logstashcrd.Logstash, data map[string]any) (bool, error) {
		if common.IsEnvtest() {
			return true, nil
		}

		sts := &appv1.StatefulSet{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "Error when read Logstash statefulset")
		}

		if sts.Status.ObservedGeneration < sts.Generation {
			return false, nil
		}

		expectedReplicas := int32(1)
		if sts.Spec.Replicas != nil {
			expectedReplicas = *sts.Spec.Replicas
		}
		if sts.Status.CurrentReplicas != expectedReplicas || sts.Status.UpdatedReplicas != expectedReplicas {
			return false, nil
		}

		return true, nil
	}
}

// cleanupLegacyLogstashCASecret deletes the pre-saga PKI secret <name>-pki-ls once
// the new saga CA secret <name>-tls-ls-ca exists. Best-effort.
func cleanupLegacyLogstashCASecret(ctx context.Context, c client.Client, o *logstashcrd.Logstash, logger *logrus.Entry) error {
	newCA := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPki(o)}, newCA); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	legacyName := fmt.Sprintf("%s-pki-ls", o.Name)
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
	logger.Infof("Deleted legacy Logstash CA secret %s/%s", o.Namespace, legacyName)

	return nil
}
