package elasticsearch

import (
	"context"
	"crypto/x509"
	"fmt"
	"regexp"
	"time"

	"github.com/disaster37/goca"
	"github.com/disaster37/goca/cert"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tlsPhase string

const (
	TlsConditionGeneratePki          = "TlsGeneratePki"
	TlsConditionPropagatePki         = "TlsPropagatePki"
	TlsConditionGenerateCertificate  = "TlsGenerateCertificates"
	TlsConditionPropagateCertificate = "TlsPropagateCertificates"
	TlsCondition                     = "TlsReady"
	TlsPhase                         = "Tls"
	DefaultRenewCertificate          = -time.Hour * 24 * 30 // 30 days before expired
)

var (
	phaseTlsCreate                tlsPhase = "tlsCreate"
	phaseTlsUpdatePki             tlsPhase = "tlsUpdatePki"
	phaseTlsPropagatePki          tlsPhase = "tlsPropagatePki"
	phaseTlsUpdateCertificates    tlsPhase = "tlsUpdateCertificates"
	phaseTlsPropagateCertificates tlsPhase = "tlsPropagateCertificates"
	phaseTlsCleanTransportCA      tlsPhase = "tlsCleanCA"
	phaseTlsNormal                tlsPhase = "tlsNormal"
)

type TlsReconciler struct {
	common.Reconciler
}

func NewTlsReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &TlsReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "tls",
			}),
			Name:   "tls",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *TlsReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, TlsCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}
	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionGeneratePki) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionGeneratePki,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionPropagatePki) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionPropagatePki,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionGenerateCertificate) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionGenerateCertificate,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionPropagateCertificate) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionPropagateCertificate,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = TlsPhase

	return res, nil
}

// Read existing transport TLS secret
func (r *TlsReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
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
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sTransportPki); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
		}
		sTransportPki = nil
	}

	// Read transport secret
	secretName = GetSecretNameForTlsTransport(o)
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sTransport); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
		}
		sTransport = nil
	}

	if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
		// Read API PKI secret
		secretName = GetSecretNameForPkiApi(o)
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApiPki); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApiPki = nil
		}

		// Read API secret
		secretName = GetSecretNameForTlsApi(o)
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApi); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApi = nil
		}
	}

	// Load transport PKI
	if sTransportPki != nil {
		// Load root CA
		transportRootCA, err = pki.LoadRootCATransport(sTransportPki.Data["ca.key"], sTransportPki.Data["ca.pub"], sTransportPki.Data["ca.crt"], sTransportPki.Data["ca.crl"], r.Log)
		if err != nil {
			return res, errors.Wrap(err, "Error when load PKI for transport layout")
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
						return res, errors.Wrapf(err, "Error when load node certificate %s", rRes[1])
					}
					nodeCertificates[rRes[1]] = *nodeCrt
				}
			}
		}
	}

	// Load API PKI
	if sApiPki != nil {
		// Load root CA
		apiRootCA, err = pki.LoadRootCATransport(sApiPki.Data["ca.key"], sApiPki.Data["ca.pub"], sApiPki.Data["ca.crt"], sApiPki.Data["ca.crl"], r.Log)
		if err != nil {
			return res, errors.Wrap(err, "Error when load PKI for API layout")
		}
	}

	// Load API certificate
	if sApi != nil {
		apiCrt, err = cert.LoadCertFromPem(sApi.Data["tls.crt"])
		if err != nil {
			return res, errors.Wrapf(err, "Error when load API certificate")
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

	return res, nil
}

// Diff permit to check if transport secrets are up to date
func (r *TlsReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var (
		d         any
		needRenew bool
		isUpdated bool
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

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}
	secretToUpdate := make([]client.Object, 0)
	secretToCreate := make([]client.Object, 0)
	secretToDelete := make([]client.Object, 0)

	// phaseInit -> phaseCreate
	// Generate all certificates
	if sTransport == nil {
		r.Log.Debugf("Detect phase: %s", phaseTlsCreate)

		diff.Diff.WriteString("Generate new certificates\n")

		// Generate transport PKI
		tmpTransportPki, transportRootCA, err := BuildTransportPkiSecret(o)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when generate transport PKI")
		}
		sTransportPki, isUpdated = updateSecret(sTransportPki, tmpTransportPki)
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransportPki); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransportPki.Name)
		}
		if isUpdated {
			secretToUpdate = append(secretToUpdate, sTransportPki)
		} else {
			err = ctrl.SetControllerReference(o, sTransportPki, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, sTransportPki)
		}

		// Generate nodes certificates
		sTransport, err = BuildTransportSecret(o, transportRootCA)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when generate nodes certificates")
		}
		err = ctrl.SetControllerReference(o, sTransport, r.Scheme)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when set as owner reference")
		}
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransport); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransport.Name)
		}
		secretToCreate = append(secretToCreate, sTransport)

		// Handle API certificates
		if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {

			// Generate API PKI
			tmpApiPki, apiRootCA, err := BuildApiPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate API PKI")
			}
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApiPki)
			}

			// Generate API certificate
			tmpApi, err := BuildApiSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate API certificate")
			}
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApi)
			} else {
				err = ctrl.SetControllerReference(o, sApi, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApi)
			}
		}

		if len(secretToCreate) > 0 {
			diff.NeedCreate = true
		}
		if len(secretToUpdate) > 0 {
			diff.NeedUpdate = true
		}
		if len(secretToDelete) > 0 {
			diff.NeedDelete = true
		}

		data["listToCreate"] = secretToCreate
		data["listToUpdate"] = secretToUpdate
		data["listToDelete"] = secretToDelete
		data["phase"] = phaseTlsCreate

		return diff, res, nil
	}

	// phaseGeneratePki -> phasePropagatePKI
	// Wait new CA propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGeneratePki, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagatePki, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseTlsPropagatePki)
		data["phase"] = phaseTlsPropagatePki
		return diff, res, nil
	}

	// phasePropagatePki -> phaseUpdateCertificates
	// Generate all certificates
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagatePki, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate, metav1.ConditionFalse) {

		r.Log.Debugf("Detect phase: %s", phaseTlsUpdateCertificates)

		// Generates certificates
		tmpTransportSecret, err := BuildTransportSecret(o, transportRootCA)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when renew transport certificates")
		}
		// Keep here the transitionnal CA appended in previous step
		tmpTransportSecret.Data["ca.crt"] = sTransport.Data["ca.crt"]
		diff.Diff.WriteString("Renew nodes certificates\n")
		sTransport, isUpdated = updateSecret(sTransport, tmpTransportSecret)
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransport); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransport.Name)
		}
		if isUpdated {
			secretToUpdate = append(secretToUpdate, sTransport)
		} else {
			err = ctrl.SetControllerReference(o, sTransport, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, sTransport)
		}

		if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {

			tmpApi, err := BuildApiSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew API certificate")
			}
			// Keep here the transitionnal CA appended in previous step
			tmpApi.Data["ca.crt"] = sApi.Data["ca.crt"]
			diff.Diff.WriteString("Renew API certificate\n")
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApi)
			} else {
				err = ctrl.SetControllerReference(o, sApi, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApi)
			}
		}

		if len(secretToCreate) > 0 {
			diff.NeedCreate = true
		}
		if len(secretToUpdate) > 0 {
			diff.NeedUpdate = true
		}
		if len(secretToDelete) > 0 {
			diff.NeedDelete = true
		}

		data["listToCreate"] = secretToCreate
		data["listToUpdate"] = secretToUpdate
		data["listToDelete"] = secretToDelete
		data["phase"] = phaseTlsUpdateCertificates

		return diff, res, nil
	}

	// phaseGenerateCertificate -> phasePropagateCertificate
	// Wait new certificates propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseTlsPropagateCertificates)

		data["phase"] = phaseTlsPropagateCertificates
		return diff, res, nil
	}

	// phaseCleanCA -> phaseNormal
	// Remove old CA certificate
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsCondition, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseTlsCleanTransportCA)

		if sTransport != nil && transportRootCA != nil {
			sTransport.Data["ca.crt"] = []byte(transportRootCA.GetCertificate())
			secretToUpdate = append(secretToUpdate, sTransport)
			diff.NeedUpdate = true
			diff.Diff.WriteString(fmt.Sprintf("Clean old ca certificate from secret %s\n", sTransport.Name))
		}

		if sApi != nil && apiRootCA != nil {
			sApi.Data["ca.crt"] = []byte(apiRootCA.GetCertificate())
			secretToUpdate = append(secretToUpdate, sApi)
			diff.NeedUpdate = true
			diff.Diff.WriteString(fmt.Sprintf("Clean old ca certificate from secret %s\n", sApi.Name))
		}

		data["listToCreate"] = secretToCreate
		data["listToUpdate"] = secretToUpdate
		data["listToDelete"] = secretToDelete
		data["phase"] = phaseTlsCleanTransportCA

		return diff, res, nil
	}

	// Check if certificates will expire or if all certicates exists (excepts node certificate)
	isRenew := false
	certificates := map[string]x509.Certificate{}
	if transportRootCA != nil {
		// No deed to recreate here, it wil generate in next
		certificates["transportPki"] = *transportRootCA.GoCertificate()
	} else {
		isRenew = true
	}
	for nodeName, nodeCrt := range nodeCertificates {
		certificates[nodeName] = nodeCrt
	}
	if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
		if apiRootCA != nil {
			certificates["apiPki"] = *apiRootCA.GoCertificate()
		} else {
			isRenew = true

			// Need to recreate Pki to allow node to start
			sApiPki, apiRootCA, err = BuildApiPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate deleted API PKI")
			}

			err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
			secretToCreate = append(secretToCreate, sApiPki)
		}
		if apiCrt != nil {
			certificates["apiCrt"] = *apiCrt
		} else {
			isRenew = true

			// Need to recreate certificate to allow node to start
			sApi, err = BuildApiSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate deleted API certificate")
			}
			err = ctrl.SetControllerReference(o, sApi, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
			secretToCreate = append(secretToCreate, sApi)

		}
	}

	if !isRenew {
		// Check certificate validity if all certificates exists
		for name, crt := range certificates {
			needRenew, err = pki.NeedRenewCertificate(&crt, defaultRenewCertificate, r.Log)
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
		r.Log.Debugf("Detect phase: %s", phaseTlsUpdatePki)

		// Renew only pki and wait all nodes get the new CA before to upgrade certificates

		// Transport PKI
		tmpTransportPki, transportRootCA, err := BuildTransportPkiSecret(o)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when renew transport PKI")
		}
		diff.Diff.WriteString("Renew transport PKI\n")

		sTransportPki, isUpdated = updateSecret(sTransportPki, tmpTransportPki)
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransportPki); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransportPki.Name)
		}
		if isUpdated {
			secretToUpdate = append(secretToUpdate, sTransportPki)
		} else {
			err = ctrl.SetControllerReference(o, sTransportPki, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, sTransportPki)
		}

		// Append new CA with others CA
		sTransport.Data["ca.crt"] = []byte(fmt.Sprintf("%s\n%s", string(sTransport.Data["ca.crt"]), transportRootCA.GetCertificate()))
		secretToUpdate = append(secretToUpdate, sTransport)

		// API PKI
		if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
			tmpApiPki, apiRootCA, err := BuildApiPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew Api PKI")
			}
			diff.Diff.WriteString("Renew Api PKI\n")

			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransportPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApiPki)
			}

			// Append new CA with others CA
			sApi.Data["ca.crt"] = []byte(fmt.Sprintf("%s\n%s", string(sApi.Data["ca.crt"]), apiRootCA.GetCertificate()))
			secretToUpdate = append(secretToUpdate, sApi)
		}

		if len(secretToCreate) > 0 {
			diff.NeedCreate = true
		}
		if len(secretToUpdate) > 0 {
			diff.NeedUpdate = true
		}
		if len(secretToDelete) > 0 {
			diff.NeedDelete = true
		}

		data["listToCreate"] = secretToCreate
		data["listToUpdate"] = secretToUpdate
		data["listToDelete"] = secretToDelete
		data["phase"] = phaseTlsUpdatePki

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
		sTransport.Annotations = getAnnotations(o)

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sTransport); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sTransport.Name)
		}
		secretToUpdate = append(secretToUpdate, sTransport)
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []*corev1.Secret{
		sTransportPki,
	}
	if len(addedNode) == 0 && len(deletedNode) == 0 {
		// Not reconcile labels and annotation for transport secret if already updated on previous step
		secrets = append(secrets, sTransport)
	}
	if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
		secrets = append(secrets, sApiPki, sApi)
	}
	for _, s := range secrets {
		isUpdated := false
		if strDiff := localhelper.DiffLabels(getLabels(o), s.Labels); strDiff != "" {
			diff.Diff.WriteString(strDiff + "\n")
			s.Labels = getLabels(o)
			isUpdated = true
		}
		if strDiff := localhelper.DiffAnnotations(getAnnotations(o), s.Annotations); strDiff != "" {
			diff.Diff.WriteString(strDiff + "\n")
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

	if len(secretToCreate) > 0 {
		diff.NeedCreate = true
	}
	if len(secretToUpdate) > 0 {
		diff.NeedUpdate = true
	}
	if len(secretToDelete) > 0 {
		diff.NeedDelete = true
	}

	data["listToCreate"] = secretToCreate
	data["listToUpdate"] = secretToUpdate
	data["listToDelete"] = secretToDelete
	data["phase"] = phaseTlsNormal

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *TlsReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	// Update main condition
	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    TlsCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *TlsReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var (
		d any
	)

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(tlsPhase)

	switch phase {
	case phaseTlsCreate:

		// Check if pod already exist. Need to delete them to have a chance to reconcile
		// It's a kind of last hope
		// Need to delete all pod to force to reconcil with right ca / certificates
		podList := &corev1.PodList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
		}
		if len(podList.Items) > 0 {
			r.Log.Warn("Transport secret have been deleted and Elasticsearch pods already exists. Start to delete them for the last hope to reconcile")
			for _, p := range podList.Items {
				if err = r.Client.Delete(ctx, &p); err != nil {
					return res, errors.Wrapf(err, "Error when delete pod %s", p.Name)
				}
				r.Log.Infof("Successfully delete pod %s", p.Name)
			}

			r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Existing pod successfully deleted")
		}

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Tls secrets successfully generated")

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGeneratePki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagatePki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "CA certificate propagated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates propagated",
		})

		r.Log.Info("Phase Create all certificates successfully finished")
	case phaseTlsUpdatePki:

		// The statefullset controller will upgrade statefullset because of the checksum certificate change
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGeneratePki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagatePki,
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait propagate new CA certificate",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate,
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait generate new certificates",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate,
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait propagate new certificates",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait renew all certificates",
		})

		r.Log.Info("Phase to renew PKI successfully finished")

	case phaseTlsPropagatePki:
		// Wait statefullset controller finished to propagate pki (finished to upgrade all statefullset)
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionFalse) {
			// all CA upgrade are finished
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:    TlsConditionPropagatePki,
				Reason:  "Success",
				Status:  metav1.ConditionTrue,
				Message: "PKI generated",
			})

			r.Log.Info("Phase propagate CA: all statefulset restarted successfully with new CA")
		} else {
			r.Log.Info("Phase propagate CA: wait statefullset controller finished to propagate CA certificate")
		}

	case phaseTlsUpdateCertificates:
		// The statefullset controller will upgrade statefullset because of the checksum certificate change
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		r.Log.Info("Phase propagate certificates: all certificates have been successfully renewed")

	case phaseTlsPropagateCertificates:
		// Wait statefullset controller finished to propagate certfificates (finished to upgrade all statefullset)
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionFalse) {

			// all certificate upgrade are finished
			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:    TlsConditionPropagateCertificate,
				Reason:  "Success",
				Status:  metav1.ConditionTrue,
				Message: "Certificates propagated",
			})

			r.Log.Info("Phase propagate certificates: all nodes have been successfully restarted with new certificates")
		} else {
			r.Log.Info("Phase propagate certificates:  wait statefullset controller finished to propagate certificate")
		}

	case phaseTlsCleanTransportCA:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})

		r.Log.Info("Clean old transport CA certificate successfully")

	}

	return res, nil
}
