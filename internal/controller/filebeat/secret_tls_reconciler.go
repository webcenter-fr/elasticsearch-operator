package filebeat

import (
	"context"
	"fmt"
	"net"
	"sort"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged/pernode"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
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
	AnnotationForceRenewTLS = "filebeat.k8s.webcenter.fr/force-renew-tls"
	// AnnotationForceRenewCertificates forces leaf-only renewal (read as "true").
	AnnotationForceRenewCertificates = "filebeat.k8s.webcenter.fr/force-renew-certificates"
)

// tlsReconciler wraps the TLS rotation saga for Filebeat self-managed PKI.
// PKI-disabled / empty-tls paths are short-circuited in Read.
type tlsReconciler struct {
	workflow.WorkflowStepReconcilerActionWithDiff[*beatcrd.Filebeat, client.Object]
}

func newTlsReconciler(c client.Client, recorder record.EventRecorder) multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, client.Object] {
	saga := rotation.NewTLSStep[*beatcrd.Filebeat](
		c,
		TlsPhase,
		TlsCondition,
		recorder,
		common.FieldManager,
		pernode.NewPerNodeBackend[*beatcrd.Filebeat](&filebeatNodeSpecProvider{}),
		certificate.TLSSpecProviderFunc[*beatcrd.Filebeat](filebeatTLSSpec),
		rotation.WithConvergenceCheck(filebeatConvergenceCheck(c)),
		rotation.WithForceRegenerateAllAnnotation[*beatcrd.Filebeat](AnnotationForceRenewTLS),
		rotation.WithForceRegenerateLeafAnnotation[*beatcrd.Filebeat](AnnotationForceRenewCertificates),
		rotation.WithLabelsDecorator(func(o *beatcrd.Filebeat, obj client.Object) {
			obj.SetLabels(getLabels(o))
		}),
		rotation.WithAnnotationsDecorator(func(o *beatcrd.Filebeat, obj client.Object) {
			obj.SetAnnotations(getAnnotations(o))
			obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
		}),
	)
	return &tlsReconciler{WorkflowStepReconcilerActionWithDiff: saga}
}

// Read short-circuits the saga when PKI is disabled or has no TLS entries;
// otherwise performs legacy-CA cleanup and delegates to the saga.
func (r *tlsReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (multiphase.MultiPhaseRead[client.Object], reconcile.Result, error) {
	if !o.Spec.Pki.IsEnabled() || len(o.Spec.Pki.Tls) == 0 {
		return multiphase.NewMultiPhaseRead[client.Object](), reconcile.Result{}, nil
	}

	if err := cleanupLegacyFilebeatCASecret(ctx, r.Client(), o, logger); err != nil {
		logger.Warnf("Failed to clean up legacy Filebeat CA secret: %s", err.Error())
	}

	return r.WorkflowStepReconcilerActionWithDiff.Read(ctx, o, data, logger)
}

// filebeatNodeSpecProvider implements pernode.NodeSpecProvider[*beatcrd.Filebeat];
// "nodes" are the keys of spec.pki.tls.
type filebeatNodeSpecProvider struct{}

func (p *filebeatNodeSpecProvider) ExpectedNodeNames(o *beatcrd.Filebeat) ([]string, error) {
	names := make([]string, 0, len(o.Spec.Pki.Tls))
	for name := range o.Spec.Pki.Tls {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (p *filebeatNodeSpecProvider) NodeCertSpec(o *beatcrd.Filebeat, nodeName string) (cn string, dnsNames []string, ips []string, err error) {
	tlsSpec, ok := o.Spec.Pki.Tls[nodeName]
	if !ok {
		return "", nil, nil, errors.Errorf("unknown TLS entry %q", nodeName)
	}

	dnsNames = make([]string, 0, (len(o.Spec.Services)*7)+len(o.Spec.Ingresses))
	for _, service := range o.Spec.Services {
		dnsNames = append(dnsNames,
			GetServiceName(o, service.Name),
			fmt.Sprintf("%s.%s", GetServiceName(o, service.Name), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetServiceName(o, service.Name), o.Namespace),
			GetGlobalServiceName(o),
			fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
			fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
			fmt.Sprintf("*.%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
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

// filebeatTLSSpec builds the shared base TLSSpec (subject, validity, renewal,
// RSA key size). Per-node CN/DNS/IPs are supplied by NodeCertSpec.
func filebeatTLSSpec(o *beatcrd.Filebeat) certificate.TLSSpec {
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
		CACommonName:     fmt.Sprintf("%s-filebeat", o.Name),
		LeafValidityDays: validityDays,
		CAValidityDays:   validityDays,
		RenewalDays:      renewalDays,
		KeyAlgorithm:     certificate.KeyAlgorithmRSA,
		KeySize:          keySize,
		Subject: certificate.CertificateSubject{
			Organizations:       []string{o.Name},
			OrganizationalUnits: []string{"filebeat"},
			Countries:           []string{"internal"},
			Localities:          []string{"internal"},
			Provinces:           []string{"internal"},
		},
	}
}

// filebeatConvergenceCheck gates Rotate→Converge on the Filebeat StatefulSet
// having rolled to the latest generation.
func filebeatConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*beatcrd.Filebeat] {
	return func(ctx context.Context, o *beatcrd.Filebeat, data map[string]any) (bool, error) {
		if common.IsEnvtest() {
			return true, nil
		}

		sts := &appv1.StatefulSet{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "Error when read Filebeat statefulset")
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

// cleanupLegacyFilebeatCASecret deletes the pre-saga PKI secret <name>-pki-fb once
// the new saga CA secret <name>-tls-fb-ca exists. Best-effort.
func cleanupLegacyFilebeatCASecret(ctx context.Context, c client.Client, o *beatcrd.Filebeat, logger *logrus.Entry) error {
	newCA := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPki(o)}, newCA); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	legacyName := fmt.Sprintf("%s-pki-fb", o.Name)
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
	logger.Infof("Deleted legacy Filebeat CA secret %s/%s", o.Namespace, legacyName)

	return nil
}
