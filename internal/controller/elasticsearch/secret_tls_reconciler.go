package elasticsearch

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/rotation"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate/selfmanaged/pernode"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/workflow"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	TlsTransportCondition shared.ConditionName = "TlsTransportReady"
	TlsTransportPhase     shared.PhaseName     = "TlsTransport"
	TlsApiCondition       shared.ConditionName = "TlsApiReady"
	TlsApiPhase           shared.PhaseName     = "TlsApi"
	TlsCondition          shared.ConditionName = "TlsReady"
	TlsConditionBlackout  shared.ConditionName = "TlsBlackout"
)

// transportNodeSpecProvider supplies per-node certificate parameters for the
// Elasticsearch transport layer. O/OU/SAN/CN/IP values are preserved from the
// previous goca-based implementation.
type transportNodeSpecProvider struct{}

func (p *transportNodeSpecProvider) ExpectedNodeNames(o *elasticsearchcrd.Elasticsearch) ([]string, error) {
	return GetNodeNames(o), nil
}

func (p *transportNodeSpecProvider) NodeCertSpec(o *elasticsearchcrd.Elasticsearch, nodeName string) (cn string, dnsNames []string, ips []string, err error) {
	nodeGroupName := GetNodeGroupNameFromNodeName(nodeName)

	cn = nodeName
	dnsNames = []string{
		fmt.Sprintf("%s.%s", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName)),
		fmt.Sprintf("%s.%s.%s", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
		fmt.Sprintf("%s.%s.%s.svc", nodeName, GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
		GetNodeGroupServiceName(o, nodeGroupName),
		fmt.Sprintf("%s.%s", GetNodeGroupServiceName(o, nodeGroupName), o.Namespace),
		fmt.Sprintf("%s.%s.svc", GetNodeGroupServiceName(o, nodeGroupName), o.Namespace),
		GetNodeGroupServiceNameHeadless(o, nodeGroupName),
		fmt.Sprintf("%s.%s", GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
		fmt.Sprintf("%s.%s.svc", GetNodeGroupServiceNameHeadless(o, nodeGroupName), o.Namespace),
		nodeName,
		fmt.Sprintf("%s.%s", nodeName, o.Namespace),
		fmt.Sprintf("%s.%s.svc", nodeName, o.Namespace),
		GetGlobalServiceName(o),
		fmt.Sprintf("%s.%s", GetGlobalServiceName(o), o.Namespace),
		fmt.Sprintf("%s.%s.svc", GetGlobalServiceName(o), o.Namespace),
	}
	ips = []string{"127.0.0.1"}

	return cn, dnsNames, ips, nil
}

// baseTLSSpec builds the shared TLSSpec skeleton (validity, renewal, subject)
// used by both the transport and API layers. Only the secret name, CN/CA-CN
// and organizational unit differ between the two.
func baseTLSSpec(o *elasticsearchcrd.Elasticsearch, secretName, commonName, caCommonName, organizationalUnit string) certificate.TLSSpec {
	validityDays := 397
	if o.Spec.Tls.ValidityDays != nil {
		validityDays = *o.Spec.Tls.ValidityDays
	}

	renewalDays := 30
	if o.Spec.Tls.RenewalDays != nil {
		renewalDays = *o.Spec.Tls.RenewalDays
	}

	spec := certificate.TLSSpec{
		SecretName:       secretName,
		CommonName:       commonName,
		CACommonName:     caCommonName,
		LeafValidityDays: validityDays,
		CAValidityDays:   validityDays,
		RenewalDays:      renewalDays,
		Subject: certificate.CertificateSubject{
			Organizations:       []string{o.Name},
			OrganizationalUnits: []string{organizationalUnit},
			Countries:           []string{"internal"},
			Localities:          []string{"internal"},
			Provinces:           []string{"internal"},
		},
	}

	applyKeyComplexity(&spec, o.Spec.Tls.KeyComplexity, o.Spec.Tls.KeySize)

	return spec
}

// transportTLSSpec computes the desired TLSSpec for the transport layer.
func transportTLSSpec(o *elasticsearchcrd.Elasticsearch) certificate.TLSSpec {
	return baseTLSSpec(o,
		GetSecretNameForTlsTransport(o),
		fmt.Sprintf("%s-transport", o.Name),
		fmt.Sprintf("%s-transport", o.Name),
		"transport",
	)
}

// applyKeyComplexity maps the operator KeyComplexity string to the library
// KeyAlgorithm / Curve / KeySize fields.
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

// transportConvergenceCheck waits until all owned StatefulSets have rolled to
// the new transport certificate before advancing the CA rotation saga.
func transportConvergenceCheck(c client.Client) rotation.ConvergenceCheck[*elasticsearchcrd.Elasticsearch] {
	return func(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any) (bool, error) {
		// In envtest there is no kubelet to roll pods, so the convergence check
		// would never pass and the CA rotation saga would stall in "Converge".
		if common.IsEnvtest() {
			return true, nil
		}

		stsList := &appv1.StatefulSetList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
		if err != nil {
			return false, errors.Wrap(err, "Error when generate label selector")
		}
		if err = c.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
			return false, errors.Wrapf(err, "Error when read statefulset")
		}

		for i := range stsList.Items {
			sts := &stsList.Items[i]
			expectedReplicas := int32(1)
			if sts.Spec.Replicas != nil {
				expectedReplicas = *sts.Spec.Replicas
			}
			if sts.Status.CurrentReplicas != expectedReplicas || sts.Status.UpdatedReplicas != expectedReplicas {
				return false, nil
			}
		}

		return true, nil
	}
}

// newTlsTransportReconciler returns the transport CA rotation saga step,
// driven by the library rotation.NewTLSStep with the selfmanaged/pernode
// backend.
func newTlsTransportReconciler(c client.Client, recorder record.EventRecorder, log *logrus.Entry) workflow.WorkflowStepReconcilerActionWithDiff[*elasticsearchcrd.Elasticsearch, client.Object] {
	return rotation.NewTLSStep[*elasticsearchcrd.Elasticsearch](
		c,
		TlsTransportPhase,
		TlsTransportCondition,
		recorder,
		common.FieldManager,
		pernode.NewPerNodeBackend[*elasticsearchcrd.Elasticsearch](&transportNodeSpecProvider{}),
		certificate.TLSSpecProviderFunc[*elasticsearchcrd.Elasticsearch](transportTLSSpec),
		rotation.WithConvergenceCheck(transportConvergenceCheck(c)),
		rotation.WithCertificateCustomizer(transportCARenewalCustomizer(c, log)),
		rotation.WithLabelsDecorator(func(o *elasticsearchcrd.Elasticsearch, obj client.Object) {
			obj.SetLabels(getLabels(o))
		}),
		rotation.WithAnnotationsDecorator(func(o *elasticsearchcrd.Elasticsearch, obj client.Object) {
			obj.SetAnnotations(getAnnotations(o))
			obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})
		}),
	)
}

// transportCARenewalCustomizer forces a CA rotation when the current transport
// CA is within caRenewalDays of expiry. The library's CANeedsRenewal only knows
// the shared RenewalDays window (now reserved for leaves), so the operator
// re-checks the CA against the CA-specific window here and, when needed, flags
// the saga to rotate via the force-regenerate annotation.
func transportCARenewalCustomizer(c client.Client, log *logrus.Entry) certificate.CertificateCustomizer[*elasticsearchcrd.Elasticsearch] {
	return certificate.CertificateCustomizerFunc[*elasticsearchcrd.Elasticsearch](func(o *elasticsearchcrd.Elasticsearch, base certificate.TLSSpec) (certificate.TLSSpec, error) {
		if o.Spec.Tls.CaRenewalDays == nil {
			return base, nil
		}

		caSecret := &corev1.Secret{}
		if err := c.Get(context.Background(), types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPkiTransport(o)}, caSecret); err != nil {
			if k8serrors.IsNotFound(err) {
				return base, nil // CA not yet generated; bootstrap creates it
			}
			return base, err
		}

		crt, err := certParse(caSecret.Data["ca.crt"])
		if err != nil {
			return base, nil // unparseable CA; avoid breaking bootstrap
		}

		needRenew, err := pki.NeedRenewCertificate(crt, time.Duration(*o.Spec.Tls.CaRenewalDays)*24*time.Hour, log)
		if err != nil || !needRenew {
			return base, nil
		}

		if o.Annotations == nil {
			o.Annotations = map[string]string{}
		}
		o.Annotations["operator-sdk-extra.webcenter.fr/force-regenerate-tls"] = "true"

		return base, nil
	})
}

// apiTlsReconciler manages the API (HTTP) TLS certificates: a single CA plus a
// single leaf certificate, regenerated on expiry. It honors the
// CertificateSecretRef (BYO) case by emitting no child objects.
type apiTlsReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]
}

func newTlsApiReconciler(c client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]) {
	return &apiTlsReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret](
			c,
			TlsApiPhase,
			TlsApiCondition,
			recorder,
			common.FieldManager,
		),
	}
}

// Read existing API TLS secrets.
func (r *apiTlsReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()

	// Best-effort cleanup of the legacy (pre-v3) transport CA secret once the
	// new saga-managed CA secret exists.
	if err := cleanupLegacyTransportCASecret(ctx, r.Client(), o, logger); err != nil {
		logger.Warnf("Failed to clean up legacy transport CA secret: %s", err.Error())
	}

	// BYO or TLS disabled: nothing to manage.
	if !o.Spec.Tls.IsTlsEnabled() || !o.Spec.Tls.IsSelfManagedSecretForTls() {
		return read, res, nil
	}

	sPki := &corev1.Secret{}
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPkiApi(o)}, sPki); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read existing secret %s", GetSecretNameForPkiApi(o))
		}
		sPki = nil
	}
	if sPki != nil {
		read.AddCurrentObject(sPki)
	}

	sApi := &corev1.Secret{}
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsApi(o)}, sApi); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read existing secret %s", GetSecretNameForTlsApi(o))
		}
		sApi = nil
	}
	if sApi != nil {
		read.AddCurrentObject(sApi)
	}

	data["apiPkiSecret"] = sPki
	data["apiTlsSecret"] = sApi

	// Compute desired API secrets. When they are up to date, expected == current
	// so that SSA apply is a no-op.
	expected, err := buildApiSecrets(r.Client(), o, sPki, sApi)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate API TLS secrets")
	}

	// The transport rotation saga manages the transport CA/leaf secrets but does
	// not re-apply labels/annotations on a label-only CR update (its "" steady
	// state registers current==expected). Detect and fix that drift here so the
	// transport secrets carry the operator labels after an update.
	transportDrift, err := buildTransportTlsMetadataDrift(ctx, r.Client(), o)
	if err != nil {
		return read, res, err
	}
	expected = append(expected, transportDrift...)

	common.InjectTypeMeta(r.Client().Scheme(), expected...)
	read.SetExpectedObjects(expected)

	return read, res, nil
}

// buildTransportTlsMetadataDrift returns copies of the transport CA and leaf
// secrets with corrected labels/annotations when they differ from the operator
// defaults (empty otherwise).
func buildTransportTlsMetadataDrift(ctx context.Context, c client.Client, o *elasticsearchcrd.Elasticsearch) ([]*corev1.Secret, error) {
	out := make([]*corev1.Secret, 0, 2)
	for _, name := range []string{GetSecretNameForPkiTransport(o), GetSecretNameForTlsTransport(o)} {
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

// cleanupLegacyTransportCASecret deletes the pre-v3 transport CA secret
// (<name>-pki-transport-es) once the new saga-managed CA secret exists. This is
// a one-time best-effort migration cleanup.
func cleanupLegacyTransportCASecret(ctx context.Context, c client.Client, o *elasticsearchcrd.Elasticsearch, logger *logrus.Entry) error {
	// The new CA must exist first (the saga has generated it at least once).
	newCA := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForPkiTransport(o)}, newCA); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	legacyName := fmt.Sprintf("%s-pki-transport-es", o.Name)
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
	logger.Infof("Deleted legacy transport CA secret %s/%s", o.Namespace, legacyName)

	return nil
}

// buildApiSecrets computes the expected API CA + leaf secrets.
func buildApiSecrets(c client.Client, o *elasticsearchcrd.Elasticsearch, sPki, sApi *corev1.Secret) ([]*corev1.Secret, error) {
	spec := apiTLSSpec(o)

	needCA, err := apiCANeedsRegeneration(o, sPki, spec)
	if err != nil {
		return nil, err
	}
	needLeaf, err := apiLeafNeedsRegeneration(sApi, spec)
	if err != nil {
		return nil, err
	}

	if !needCA && !needLeaf {
		// Up to date: expected == current. Detect label/annotation drift so a
		// label-only CR update re-applies the secrets (SSA no-op otherwise).
		out := make([]*corev1.Secret, 0, 2)
		for _, s := range []*corev1.Secret{sPki, sApi} {
			if s == nil {
				continue
			}
			if drifted := common.DriftSecretForMetadata(s, getLabels(o), getAnnotations(o)); drifted != nil {
				out = append(out, drifted)
			} else {
				out = append(out, s.DeepCopy())
			}
		}
		return out, nil
	}

	var caSecret *corev1.Secret
	var signer crypto.Signer
	var caCert *x509.Certificate

	if needCA {
		caSecret, signer, caCert, err = selfmanaged.BuildCASigner(o.Namespace, GetSecretNameForPkiApi(o), spec)
		if err != nil {
			return nil, errors.Wrap(err, "Error when generate API PKI")
		}

		// BuildCASigner names the CA secret "<name>-ca"; re-align to the operator
		// naming convention and decorate it.
		caSecret.Name = GetSecretNameForPkiApi(o)
		caSecret.Labels = getLabels(o)
		caSecret.Annotations = getAnnotations(o)
		caSecret.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
	} else {
		// Leaf-only renewal: re-sign the leaf with the existing CA instead of
		// minting a fresh CA (which would rotate the CA and break consumers
		// still trusting the old one). sPki is non-nil here because needCA is
		// true whenever sPki is nil.
		caCert, signer, err = selfmanaged.ParseCASigner(sPki)
		if err != nil {
			return nil, errors.Wrap(err, "Error when parse existing API PKI")
		}
		caSecret = sPki.DeepCopy()
		caSecret.Labels = getLabels(o)
		caSecret.Annotations = getAnnotations(o)
		caSecret.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
	}

	leafSpec := apiTLSSpec(o)
	leafSpec.CommonName = o.Name
	leafPEM, keyPEM, err := selfmanaged.SignLeafSigner(caCert, signer, leafSpec)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate API certificate")
	}

	leafSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForTlsApi(o),
			Namespace:   o.Namespace,
			Labels:      getLabels(o),
			Annotations: getAnnotations(o),
		},
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		Type:     corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt":  caSecret.Data[selfmanaged.CAKey],
			"tls.crt": leafPEM,
			"tls.key": keyPEM,
		},
	}

	// Set owner reference on expected objects before SSA apply.
	for _, s := range []*corev1.Secret{caSecret, leafSecret} {
		if err = ctrl.SetControllerReference(o, s, c.Scheme()); err != nil {
			return nil, errors.Wrapf(err, "Error when set owner reference on object '%s'", s.GetName())
		}
	}

	return []*corev1.Secret{caSecret, leafSecret}, nil
}

func apiTLSSpec(o *elasticsearchcrd.Elasticsearch) certificate.TLSSpec {
	spec := baseTLSSpec(o,
		GetSecretNameForTlsApi(o),
		o.Name,
		fmt.Sprintf("%s-api", o.Name),
		"api",
	)

	if o.Spec.Tls.SelfSignedCertificate != nil {
		spec.DNSNames = append(spec.DNSNames, o.Spec.Tls.SelfSignedCertificate.AltNames...)
		spec.IPAddresses = append(spec.IPAddresses, o.Spec.Tls.SelfSignedCertificate.AltIps...)
	}

	return spec
}

// apiCANeedsRegeneration returns true when the API CA secret is missing or
// needs renewal (against the CA-specific caRenewalDays window).
func apiCANeedsRegeneration(o *elasticsearchcrd.Elasticsearch, sPki *corev1.Secret, spec certificate.TLSSpec) (bool, error) {
	if sPki == nil {
		return true, nil
	}

	// The library's CANeedsRenewal uses the shared RenewalDays window (reserved
	// for leaves); apply the CA-specific caRenewalDays window for the API CA.
	caSpec := spec
	if o.Spec.Tls.CaRenewalDays != nil {
		caSpec.RenewalDays = *o.Spec.Tls.CaRenewalDays
	}

	need, err := selfmanaged.CANeedsRenewal(sPki, caSpec, time.Now())
	if err != nil {
		return false, errors.Wrap(err, "Error when check API CA renewal")
	}
	if need {
		return true, nil
	}
	changed, err := selfmanaged.CAContentChanged(sPki, spec)
	if err != nil {
		return false, errors.Wrap(err, "Error when check API CA content")
	}
	return changed, nil
}

// apiLeafNeedsRegeneration returns true when the API leaf certificate is
// missing, expiring, or has drifted from the desired CN/subject/key/SAN/IPs.
func apiLeafNeedsRegeneration(sApi *corev1.Secret, spec certificate.TLSSpec) (bool, error) {
	if sApi == nil {
		return true, nil
	}
	raw, ok := sApi.Data["tls.crt"]
	if !ok || len(raw) == 0 {
		return true, nil
	}
	crt, err := certParse(raw)
	if err != nil {
		return false, errors.Wrap(err, "Error when parse API certificate")
	}

	// 1. Expiry.
	if time.Now().After(crt.NotAfter.Add(-(time.Duration(certificate.GetValidRenewalDays(spec)) * 24 * time.Hour))) {
		return true, nil
	}
	// 2. CN + subject drift.
	if crt.Subject.CommonName != spec.CommonName {
		return true, nil
	}
	if !selfmanaged.OrganizationsEqual(spec, crt) {
		return true, nil
	}
	if !selfmanaged.SubjectRestEqual(spec.LeafSubject(), crt.Subject) {
		return true, nil
	}
	// 3. Key algorithm/size drift (e.g. KeyComplexity change).
	if changed, err := selfmanaged.KeyChanged(spec, crt); err != nil {
		return false, err
	} else if changed {
		return true, nil
	}
	// 4. SAN drift (DNS + IP).
	if !stringSetEqual(spec.DNSNames, crt.DNSNames) {
		return true, nil
	}
	if !ipSANsEqual(spec.IPAddresses, crt.IPAddresses) {
		return true, nil
	}
	return false, nil
}

// stringSetEqual reports whether two string slices contain the same elements
// ignoring order and duplicates.
func stringSetEqual(a, b []string) bool {
	return setEqual(certificate.DedupStrings(a), certificate.DedupStrings(b))
}

// ipSANsEqual reports whether the desired IP strings and the certificate's IP
// SANs contain the same addresses, ignoring order and using the canonical
// net.IP form.
func ipSANsEqual(specIPs []string, certIPs []net.IP) bool {
	specCanonical := make([]string, 0, len(specIPs))
	for _, s := range specIPs {
		if ip := net.ParseIP(s); ip != nil {
			specCanonical = append(specCanonical, ip.String())
		} else {
			specCanonical = append(specCanonical, s)
		}
	}
	certCanonical := make([]string, 0, len(certIPs))
	for _, ip := range certIPs {
		certCanonical = append(certCanonical, ip.String())
	}
	return setEqual(certificate.DedupStrings(specCanonical), certificate.DedupStrings(certCanonical))
}

// setEqual reports whether two string slices contain the same set of elements.
func setEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]struct{}, len(a))
	for _, v := range a {
		m[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := m[v]; !ok {
			return false
		}
	}
	return true
}

func certParse(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("no certificate found in PEM data")
	}
	return x509.ParseCertificate(block.Bytes)
}
