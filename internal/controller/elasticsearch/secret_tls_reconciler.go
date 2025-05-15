package elasticsearch

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
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	TlsConditionGeneratePki          shared.ConditionName = "TlsGeneratePki"
	TlsConditionPropagatePki         shared.ConditionName = "TlsPropagatePki"
	TlsConditionGenerateCertificate  shared.ConditionName = "TlsGenerateCertificates"
	TlsConditionPropagateCertificate shared.ConditionName = "TlsPropagateCertificates"
	TlsCondition                     shared.ConditionName = "TlsReady"
	TlsConditionBlackout             shared.ConditionName = "TlsBlackout"
	TlsPhase                         shared.PhaseName     = "Tls"
	TlsPhaseCreate                   shared.PhaseName     = "tlsCreate"
	TlsPhaseUpdatePki                shared.PhaseName     = "tlsUpdatePki"
	TlsPhasePropagatePki             shared.PhaseName     = "tlsPropagatePki"
	TlsPhaseUpdateCertificates       shared.PhaseName     = "tlsUpdateCertificates"
	TlsPhasePropagateCertificates    shared.PhaseName     = "tlsPropagateCertificates"
	TlsPhaseCleanTransportCA         shared.PhaseName     = "tlsCleanCA"
	TlsPhaseNormal                   shared.PhaseName     = "tlsNormal"
	TlsPhaseReconcile                shared.PhaseName     = "tlsReconcile"
	DefaultRenewCertificate                               = -time.Hour * 24 * 30 // 30 days before expired
)

type tlsReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]
}

func newTlsReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret]) {
	return &tlsReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Secret](
			client,
			TlsPhase,
			TlsCondition,
			recorder,
		),
	}
}

// Configure permit to init condition
func (r *tlsReconciler) Configure(ctx context.Context, req reconcile.Request, o *elasticsearchcrd.Elasticsearch, logger *logrus.Entry) (res reconcile.Result, err error) {

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionGeneratePki.String()) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionGeneratePki.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionPropagatePki.String()) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionPropagatePki.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionGenerateCertificate.String()) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionGenerateCertificate.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionPropagateCertificate.String()) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionPropagateCertificate.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionBlackout.String()) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionBlackout.String(),
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	return r.MultiPhaseStepReconcilerAction.Configure(ctx, req, o, logger)
}

// Read existing transport TLS secret
func (r *tlsReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
	sTransport := &corev1.Secret{}
	sTransportPki := &corev1.Secret{}
	sApi := &corev1.Secret{}
	sApiPki := &corev1.Secret{}
	nodeCertificates := map[string]x509.Certificate{}
	var (
		transportRootCA *goca.CA
		apiRootCA       *goca.CA
		apiCrt          *x509.Certificate
		secretName      string
	)

	// Read transport PKI secret
	secretName = GetSecretNameForPkiTransport(o)
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sTransportPki); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
		}
		sTransportPki = nil
	}

	// Read transport secret
	secretName = GetSecretNameForTlsTransport(o)
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sTransport); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
		}
		sTransport = nil
	}

	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		// Read API PKI secret
		secretName = GetSecretNameForPkiApi(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApiPki); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApiPki = nil
		}

		// Read API secret
		secretName = GetSecretNameForTlsApi(o)
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApi); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApi = nil
		}
	}

	// Load transport PKI
	if sTransportPki != nil {
		// Load root CA
		transportRootCA, err = pki.LoadRootCA(sTransportPki.Data["ca.key"], sTransportPki.Data["ca.pub"], sTransportPki.Data["ca.crt"], sTransportPki.Data["ca.crl"], logger)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when load PKI for transport layout")
		}
	}

	// Load nodes certificates
	if sTransport != nil {
		// Load node certificates
		r := regexp.MustCompile(`^(.*)\.crt$`)
		for key, value := range sTransport.Data {
			if key != "ca.crt" {
				rRes := r.FindStringSubmatch(key)
				if len(rRes) > 1 {
					nodeCrt, err := cert.LoadCertFromPem(value)
					if err != nil {
						return read, res, errors.Wrapf(err, "Error when load node certificate %s", rRes[1])
					}
					nodeCertificates[rRes[1]] = *nodeCrt
				}
			}
		}
	}

	// Load API PKI
	if sApiPki != nil {
		// Load root CA
		apiRootCA, err = pki.LoadRootCA(sApiPki.Data["ca.key"], sApiPki.Data["ca.pub"], sApiPki.Data["ca.crt"], sApiPki.Data["ca.crl"], logger)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when load PKI for API layout")
		}
	}

	// Load API certificate
	if sApi != nil {
		apiCrt, err = cert.LoadCertFromPem(sApi.Data["tls.crt"])
		if err != nil {
			return read, res, errors.Wrapf(err, "Error when load API certificate")
		}
	}

	data["transportRootCA"] = transportRootCA
	data["nodeCertificates"] = nodeCertificates
	data["apiRootCA"] = apiRootCA
	data["apiCertificate"] = apiCrt
	data["transportTlsSecret"] = sTransport
	data["transportPkiSecret"] = sTransportPki
	data["apiTlsSecret"] = sApi
	data["apiPkiSecret"] = sApiPki

	return read, res, nil
}

// Diff permit to check if transport secrets are up to date
func (r *tlsReconciler) Diff(ctx context.Context, o *elasticsearchcrd.Elasticsearch, read multiphase.MultiPhaseRead[*corev1.Secret], data map[string]any, logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff multiphase.MultiPhaseDiff[*corev1.Secret], res reconcile.Result, err error) {
	var (
		d                  any
		needRenew          bool
		isUpdated          bool
		isBlackout         bool
		isClusterBootstrap bool
	)

	defaultRenewCertificate := DefaultRenewCertificate
	if o.Spec.Tls.RenewalDays != nil {
		defaultRenewCertificate = time.Duration(*o.Spec.Tls.RenewalDays) * 24 * time.Hour
	}

	d, err = helper.Get(data, "transportRootCA")
	if err != nil {
		return diff, res, err
	}
	transportRootCA := d.(*goca.CA)

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

	d, err = helper.Get(data, "nodeCertificates")
	if err != nil {
		return diff, res, err
	}
	nodeCertificates := d.(map[string]x509.Certificate)

	d, err = helper.Get(data, "transportPkiSecret")
	if err != nil {
		return diff, res, err
	}
	sTransportPki := d.(*corev1.Secret)

	d, err = helper.Get(data, "transportTlsSecret")
	if err != nil {
		return diff, res, err
	}
	sTransport := d.(*corev1.Secret)

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

	diff = multiphase.NewMultiPhaseDiff[*corev1.Secret]()

	data["isTlsBlackout"] = false

	// Check if on blackout
	// When cluster no yet bootstrapping, the certificates not yet exist
	if sTransport == nil || sTransportPki == nil || transportRootCA == nil || transportRootCA.GoCertificate().NotAfter.Before(time.Now()) {
		if !o.IsBoostrapping() {
			isClusterBootstrap = true
		} else {
			isBlackout = true
		}
	}
	for _, nodeCrt := range nodeCertificates {
		if nodeCrt.NotAfter.Before(time.Now()) {
			isBlackout = true
			break
		}
	}

	// Generate all certificates when bootstrap cluster or when we are on blackout
	if isClusterBootstrap || isBlackout {
		logger.Debugf("Detect phase: %s", TlsPhaseCreate)

		diff.AddDiff("Generate new certificates")
		secretToUpdate := make([]*corev1.Secret, 0)
		secretToCreate := make([]*corev1.Secret, 0)
		if secretToCreate, secretToUpdate, err = r.generateAllSecretsCertificates(o, sTransportPki, sTransport, sApiPki, sApi, secretToCreate, secretToUpdate); err != nil {
			return diff, res, err
		}

		diff.SetObjectsToCreate(secretToCreate)
		diff.SetObjectsToUpdate(secretToUpdate)
		data["phase"] = TlsPhaseCreate
		data["isTlsBlackout"] = isBlackout

		return diff, res, nil
	}

	// Require API secret exist to allow pod start
	// Not real blackout because of not need to restart all pod on same time
	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		isBad := false
		if apiRootCA == nil {
			// Generate API PKI
			sApiPki, apiRootCA, isUpdated, err = r.generateAPISecretPki(o, sApiPki)
			if err != nil {
				return diff, res, err
			}
			if !isUpdated {
				diff.AddObjectToCreate(sApiPki)
			} else {
				diff.AddObjectToUpdate(sApiPki)
			}

			diff.AddDiff("Generate new API PKI")

			isBad = true
		}
		if apiCrt == nil {

			// Generate API certificate
			sApi, isUpdated, err = r.generateApiSecretCertificate(o, sApi, apiRootCA)
			if err != nil {
				return diff, res, err
			}
			if !isUpdated {
				diff.AddObjectToCreate(sApi)
			} else {
				diff.AddObjectToUpdate(sApi)
			}

			diff.AddDiff("Generate new API certificate")

			isBad = true
		}

		if isBad {
			data["phase"] = TlsPhaseReconcile
			return diff, res, nil
		}
	}

	// phaseGeneratePki -> phasePropagatePKI
	// Wait new CA propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGeneratePki.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagatePki.String(), metav1.ConditionFalse) {
		logger.Debugf("Detect phase: %s", TlsPhasePropagatePki)
		data["phase"] = TlsPhasePropagatePki
		return diff, res, nil
	}

	// phasePropagatePki -> phaseUpdateCertificates
	// Generate all certificates
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagatePki.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate.String(), metav1.ConditionFalse) {

		logger.Debugf("Detect phase: %s", TlsPhaseUpdateCertificates)

		// Generate nodes certificates
		tmpTransport, isUpdated, err := r.generateTransportSecretCertificates(o, sTransport, transportRootCA)
		if err != nil {
			return diff, res, err
		}

		// Keep transitional CA
		tmpTransport.Data["ca.crt"] = sTransport.Data["ca.crt"]

		sTransport = tmpTransport
		if !isUpdated {
			diff.AddObjectToCreate(sTransport)
		} else {
			diff.AddObjectToUpdate(sTransport)
		}

		diff.AddDiff("Generate new transport certificates")

		if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {

			// Generate API certificate
			tmpApi, isUpdated, err := r.generateApiSecretCertificate(o, sApi, apiRootCA)
			if err != nil {
				return diff, res, err
			}
			// Keep transisional CA
			tmpApi.Data["ca.crt"] = sApi.Data["ca.crt"]

			sApi = tmpApi
			if !isUpdated {
				diff.AddObjectToCreate(sApi)
			} else {
				diff.AddObjectToUpdate(sApi)
			}

			diff.AddDiff("Generate new API certificate")
		}

		data["phase"] = TlsPhaseUpdateCertificates

		return diff, res, nil
	}

	// phaseGenerateCertificate -> phasePropagateCertificate
	// Wait new certificates propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate.String(), metav1.ConditionFalse) {
		logger.Debugf("Detect phase: %s", TlsPhasePropagateCertificates)

		data["phase"] = TlsPhasePropagateCertificates
		return diff, res, nil
	}

	// phaseCleanCA -> phaseNormal
	// Remove old CA certificate
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsCondition.String(), metav1.ConditionFalse) {
		logger.Debugf("Detect phase: %s", TlsPhaseCleanTransportCA)

		if sTransport != nil && transportRootCA != nil {
			sTransport.Data["ca.crt"] = []byte(transportRootCA.GetCertificate())
			diff.AddObjectToUpdate(sTransport)
			diff.AddDiff(fmt.Sprintf("Clean old ca certificate from secret %s", sTransport.Name))
		}

		if sApi != nil && apiRootCA != nil {
			sApi.Data["ca.crt"] = []byte(apiRootCA.GetCertificate())
			diff.AddObjectToUpdate(sApi)
			diff.AddDiff(fmt.Sprintf("Clean old ca certificate from secret %s", sApi.Name))
		}

		data["phase"] = TlsPhaseCleanTransportCA

		return diff, res, nil
	}

	// Check if certificates will expire or if all certicates exists (excepts node certificate)
	isRenew := false

	// Force renew certificate by annotation
	if o.Annotations[fmt.Sprintf("%s/renew-certificates", elasticsearchcrd.ElasticsearchAnnotationKey)] == "true" {
		logger.Info("Force renew certificat by annotation")
		isRenew = true
	}

	certificates := map[string]x509.Certificate{
		"transportPki": *transportRootCA.GoCertificate(),
	}

	for nodeName, nodeCrt := range nodeCertificates {
		certificates[nodeName] = nodeCrt
	}

	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		certificates["apiPki"] = *apiRootCA.GoCertificate()
		certificates["apiCrt"] = *apiCrt
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
		logger.Debugf("Detect phase: %s", TlsPhaseUpdatePki)
		// Renew only pki and wait all nodes get the new CA before to upgrade certificates

		// Generate transport PKI
		sTransportPki, transportRootCA, isUpdated, err := r.generateTransportSecretPki(o, sTransportPki)
		if err != nil {
			return diff, res, err
		}
		if !isUpdated {
			diff.AddObjectToCreate(sTransportPki)
		} else {
			diff.AddObjectToUpdate(sTransportPki)
		}
		diff.AddDiff("Renew transport PKI")

		// Append new CA with others CA and change sequence to propagate it on pod (rolling restart)
		sTransport.Data["ca.crt"] = []byte(fmt.Sprintf("%s\n%s", string(sTransport.Data["ca.crt"]), transportRootCA.GetCertificate()))
		sTransport.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)] = helper.RandomString(64)
		diff.AddObjectToUpdate(sTransport)

		// API PKI
		if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
			// Generate API PKI
			sApiPki, apiRootCA, isUpdated, err := r.generateAPISecretPki(o, sApiPki)
			if err != nil {
				return diff, res, err
			}
			if !isUpdated {
				diff.AddObjectToCreate(sApiPki)
			} else {
				diff.AddObjectToUpdate(sApiPki)
			}
			diff.AddDiff("Renew Api PKI")

			// Append new CA with others CA and change sequence to propagate it on pod (rolling restart)
			sApi.Data["ca.crt"] = []byte(fmt.Sprintf("%s\n%s", string(sApi.Data["ca.crt"]), apiRootCA.GetCertificate()))
			sApi.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)] = helper.RandomString(64)
			diff.AddObjectToUpdate(sApi)
		}

		data["phase"] = TlsPhaseUpdatePki

		return diff, res, nil
	}

	// Check if need to add or remove node certifificate
	addedNode, deletedNode := funk.DifferenceString(GetNodeNames(o), funk.Keys(nodeCertificates).([]string))
	for _, nodeName := range addedNode {
		// Generate new node certificate without rolling upgrade other nodes
		nodeCrt, err := generateNodeCertificate(o, GetNodeGroupNameFromNodeName(nodeName), nodeName, transportRootCA)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when generate node certificate for %s", nodeName)
		}
		sTransport.Data[fmt.Sprintf("%s.crt", nodeName)] = []byte(nodeCrt.Certificate)
		sTransport.Data[fmt.Sprintf("%s.key", nodeName)] = []byte(nodeCrt.RsaPrivateKey)
	}

	for _, nodeName := range deletedNode {
		// Remove entry on secret
		delete(sTransport.Data, fmt.Sprintf("%s.crt", nodeName))
		delete(sTransport.Data, fmt.Sprintf("%s.key", nodeName))
	}

	if len(addedNode) > 0 || len(deletedNode) > 0 {
		sTransport.Labels = getLabels(o)
		// Keep existing sequence to not rolling restart all nodes
		sTransport.Annotations = getAnnotations(o, map[string]string{
			fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey): sTransport.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)],
		})

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransport); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransport.Name)
		}
		diff.AddObjectToUpdate(sTransport)
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []*corev1.Secret{
		sTransportPki,
	}
	if len(addedNode) == 0 && len(deletedNode) == 0 {
		// Not reconcile labels and annotation for transport secret if already updated on previous step
		secrets = append(secrets, sTransport)
	}
	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {
		secrets = append(secrets, sApiPki, sApi)
	}
	for _, s := range secrets {
		isUpdated := false
		if s.Name == GetSecretNameForTlsApi(o) {
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
		strDiff, err := helper.DiffOwnerReferences(o, s)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when diff owner references on secret %s", s.Name)
		}
		if strDiff != "" {
			diff.AddDiff(strDiff)
			// Set ownerReferences
			if err = ctrl.SetControllerReference(o, s, r.Client().Scheme()); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", s.GetName())
			}
			isUpdated = true
		}

		if isUpdated {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(s); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", s.Name)
			}
			diff.AddObjectToUpdate(s)
		}
	}

	data["phase"] = TlsPhaseNormal

	return diff, res, nil
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *tlsReconciler) OnSuccess(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, diff multiphase.MultiPhaseDiff[*corev1.Secret], logger *logrus.Entry) (res reconcile.Result, err error) {
	var d any

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(shared.PhaseName)

	d, err = helper.Get(data, "isTlsBlackout")
	if err != nil {
		return res, err
	}
	isTlsBlackout := d.(bool)

	logger.Debugf("TLS phase : %s, isBlackout: %t", phase, isTlsBlackout)

	if isTlsBlackout {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionFalse) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:    TlsConditionBlackout.String(),
				Reason:  "Blackout",
				Status:  metav1.ConditionTrue,
				Message: "Force renew all transport certificates",
			})
		}
	} else {
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:    TlsConditionBlackout.String(),
				Reason:  "NoBlackout",
				Status:  metav1.ConditionFalse,
				Message: "Note in blackout",
			})
		}
	}

	switch phase {
	case TlsPhaseCreate:

		r.Recorder().Eventf(o, corev1.EventTypeNormal, "Completed", "Tls secrets successfully generated")

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGeneratePki.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagatePki.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "CA certificate propagated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates propagated",
		})

		logger.Info("Phase Create all certificates successfully finished")
	case TlsPhaseUpdatePki:
		// Remove force renew certificate
		if o.Annotations[fmt.Sprintf("%s/renew-certificates", elasticsearchcrd.ElasticsearchAnnotationKey)] == "true" {
			delete(o.Annotations, fmt.Sprintf("%s/renew-certificates", elasticsearchcrd.ElasticsearchAnnotationKey))
			if err = r.Client().Update(ctx, o); err != nil {
				return res, err
			}
		}

		// The statefullset controller will upgrade statefullset because of the checksum certificate change
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGeneratePki.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagatePki.String(),
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait propagate new CA certificate",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate.String(),
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait generate new certificates",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate.String(),
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait propagate new certificates",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition.String(),
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait renew all certificates",
		})

		logger.Info("Phase to renew PKI successfully finished")

	case TlsPhasePropagatePki:

		// Compute expected checksum
		d, err = helper.Get(data, "transportTlsSecret")
		if err != nil {
			return res, err
		}
		sTransport := d.(*corev1.Secret)
		sequence := sTransport.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)]

		stsList := &appv1.StatefulSetList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client().List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
			return res, errors.Wrapf(err, "Error when read statefulset")
		}

		for _, sts := range stsList.Items {
			if sts.Spec.Template.Annotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, sTransport.Name)] != sequence || localhelper.IsOnStatefulSetUpgradeState(&sts) {
				logger.Debugf("Expected: %s, actual: %s", sequence, sts.Spec.Template.Annotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, sTransport.Name)])
				logger.Info("Phase propagate CA: wait statefullset controller finished to propagate CA certificate")
				return res, nil
			}
		}

		// all CA upgrade are finished
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagatePki.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "PKI generated",
		})

		logger.Info("Phase propagate CA: all statefulset restarted successfully with new CA")
	case TlsPhaseUpdateCertificates:
		// The statefullset controller will upgrade statefullset because of the checksum certificate change
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		logger.Info("Phase propagate certificates: all certificates have been successfully renewed")

	case TlsPhasePropagateCertificates:
		// Compute expected checksum
		d, err = helper.Get(data, "transportTlsSecret")
		if err != nil {
			return res, err
		}
		sTransport := d.(*corev1.Secret)
		sequence := sTransport.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)]

		stsList := &appv1.StatefulSetList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client().List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
			return res, errors.Wrapf(err, "Error when read statefulset")
		}

		for _, sts := range stsList.Items {
			if sts.Spec.Template.Annotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, sTransport.Name)] != sequence || localhelper.IsOnStatefulSetUpgradeState(&sts) {
				logger.Debugf("Expected: %s, actual: %s", sequence, sts.Spec.Template.Annotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, sTransport.Name)])
				logger.Info("Phase propagate certificates:  wait statefullset controller finished to propagate certificate")
				return res, nil
			}
		}

		// all certificate upgrade are finished
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates propagated",
		})

		logger.Info("Phase propagate certificates: all nodes have been successfully restarted with new certificates")

	case TlsPhaseCleanTransportCA:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})

		logger.Info("Clean old transport CA certificate successfully")

	case TlsPhaseReconcile:
		return reconcile.Result{Requeue: true}, nil
	}

	return res, nil
}

func (r *tlsReconciler) generateTransportSecretPki(o *elasticsearchcrd.Elasticsearch, sTransportPki *corev1.Secret) (sTransportPkiRes *corev1.Secret, transportRootCA *goca.CA, isUpdated bool, err error) {
	tmpTransportPki, transportRootCA, err := buildTransportPkiSecret(o)
	if err != nil {
		return nil, nil, isUpdated, errors.Wrap(err, "Error when generate transport PKI")
	}
	sTransportPkiRes, isUpdated, err = updateSecret(o, sTransportPki, tmpTransportPki, r.Client().Scheme())
	if err != nil {
		return nil, nil, isUpdated, errors.Wrap(err, "Error when update secret of transport PKI")
	}
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransportPkiRes); err != nil {
		return nil, nil, isUpdated, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransportPkiRes.Name)
	}

	return sTransportPkiRes, transportRootCA, isUpdated, nil
}

func (r *tlsReconciler) generateTransportSecretCertificates(o *elasticsearchcrd.Elasticsearch, sTransport *corev1.Secret, transportRootCA *goca.CA) (sTransportRes *corev1.Secret, isUpdated bool, err error) {
	tmpTransport, err := buildTransportSecret(o, transportRootCA)
	if err != nil {
		return nil, isUpdated, errors.Wrap(err, "Error when generate nodes certificates")
	}
	sTransportRes, isUpdated, err = updateSecret(o, sTransport, tmpTransport, r.Client().Scheme())
	if err != nil {
		return nil, isUpdated, errors.Wrap(err, "Error when update secret of nodes certificates")
	}
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransportRes); err != nil {
		return nil, isUpdated, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransportRes.Name)
	}

	return sTransportRes, isUpdated, nil
}

func (r *tlsReconciler) generateAPISecretPki(o *elasticsearchcrd.Elasticsearch, sApiPki *corev1.Secret) (sApiPkiRes *corev1.Secret, apiRootCA *goca.CA, isUpdated bool, err error) {
	tmpApiPki, apiRootCA, err := buildApiPkiSecret(o)
	if err != nil {
		return nil, nil, isUpdated, errors.Wrap(err, "Error when generate API PKI")
	}
	sApiPkiRes, isUpdated, err = updateSecret(o, sApiPki, tmpApiPki, r.Client().Scheme())
	if err != nil {
		return nil, nil, isUpdated, errors.Wrap(err, "Error when update secret of API PKI")
	}
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPkiRes); err != nil {
		return nil, nil, isUpdated, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPkiRes.Name)
	}

	return sApiPkiRes, apiRootCA, isUpdated, nil
}

func (r *tlsReconciler) generateApiSecretCertificate(o *elasticsearchcrd.Elasticsearch, sApi *corev1.Secret, apiRootCA *goca.CA) (sApiRes *corev1.Secret, isUpdated bool, err error) {
	tmpApi, err := buildApiSecret(o, apiRootCA)
	if err != nil {
		return nil, isUpdated, errors.Wrap(err, "Error when generate API certificate")
	}
	sApiRes, isUpdated, err = updateSecret(o, sApi, tmpApi, r.Client().Scheme())
	if err != nil {
		return nil, isUpdated, errors.Wrap(err, "Error when update secret of API certificate")
	}
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiRes); err != nil {
		return nil, isUpdated, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiRes.Name)
	}

	return sApiRes, isUpdated, nil
}

func (r *tlsReconciler) generateAllSecretsCertificates(o *elasticsearchcrd.Elasticsearch, sTransportPki *corev1.Secret, sTransport *corev1.Secret, sApiPki *corev1.Secret, sApi *corev1.Secret, secretToCreate []*corev1.Secret, secretToUpdate []*corev1.Secret) (secretToCreateRes []*corev1.Secret, secretToUpdateRes []*corev1.Secret, err error) {
	// Generate transport PKI
	sTransportPki, transportRootCA, isUpdated, err := r.generateTransportSecretPki(o, sTransportPki)
	if err != nil {
		return secretToCreate, secretToUpdate, err
	}
	if !isUpdated {
		secretToCreate = append(secretToCreate, sTransportPki)
	} else {
		secretToUpdate = append(secretToUpdate, sTransportPki)
	}

	// Generate nodes certificates
	sTransport, isUpdated, err = r.generateTransportSecretCertificates(o, sTransport, transportRootCA)
	if err != nil {
		return secretToCreate, secretToUpdate, err
	}
	if !isUpdated {
		secretToCreate = append(secretToCreate, sTransport)
	} else {
		secretToUpdate = append(secretToUpdate, sTransport)
	}

	// Handle API certificates
	if o.Spec.Tls.IsTlsEnabled() && o.Spec.Tls.IsSelfManagedSecretForTls() {

		// Generate API PKI
		sApiPki, apiRootCA, isUpdated, err := r.generateAPISecretPki(o, sApiPki)
		if err != nil {
			return secretToCreate, secretToUpdate, err
		}
		if !isUpdated {
			secretToCreate = append(secretToCreate, sApiPki)
		} else {
			secretToUpdate = append(secretToUpdate, sApiPki)
		}

		// Generate API certificate
		sApi, isUpdated, err = r.generateApiSecretCertificate(o, sApi, apiRootCA)
		if err != nil {
			return secretToCreate, secretToUpdate, err
		}
		if !isUpdated {
			secretToCreate = append(secretToCreate, sApi)
		} else {
			secretToUpdate = append(secretToUpdate, sApi)
		}

	}

	return secretToCreate, secretToUpdate, nil
}
