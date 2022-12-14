package elasticsearch

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/disaster37/goca"
	"github.com/disaster37/goca/cert"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	podutil "k8s.io/kubectl/pkg/util/podutils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tlsPhase string

const (
	TlsConditionGenerateTransportPki  = "TlsGenerateTransportPki"
	TlsConditionPropagateTransportPki = "TlsPropagateTransportPki"
	TlsConditionGenerateCertificate   = "TlsGenerateCertificates"
	TlsConditionPropagateCertificate  = "TlsPropagateCertificates"
	TlsCondition                      = "TlsReady"
	TlsPhase                          = "Generate certificates"
	DefaultRenewCertificate           = -time.Hour * 24 * 7 // 7 days before expired
)

var (
	phaseCreate                tlsPhase = "create"
	phaseUpdateTransportPki    tlsPhase = "updateTransportPki"
	phasePropagateTransportPki tlsPhase = "propagateTransportPki"
	phaseUpdateCertificates    tlsPhase = "updateCertificates"
	phasePropagateCertificates tlsPhase = "propagateCertificates"
	phaseCleanTransportCA      tlsPhase = "cleanTransportCA"
	phaseNormal                tlsPhase = "normal"
)

type TlsReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewTlsReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sReconciler {
	return &TlsReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "tls",
	}
}

// Name return the current phase
func (r *TlsReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *TlsReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, TlsCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = TlsPhase
	}
	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionGenerateTransportPki) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionGenerateTransportPki,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, TlsConditionPropagateTransportPki) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsConditionPropagateCertificate,
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

	return res, nil
}

// Read existing transport TLS secret
func (r *TlsReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
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
		for _, nodeName := range GetNodeNames(o) {
			if sTransport.Data[fmt.Sprintf("%s.crt", nodeName)] != nil {
				nodeCrt, err := cert.LoadCertFromPem(sTransport.Data[fmt.Sprintf("%s.crt", nodeName)])
				if err != nil {
					return res, errors.Wrapf(err, "Error when load node certificate %s", nodeName)
				}
				nodeCertificates[nodeName] = *nodeCrt
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

// Create will create TLS secrets
func (r *TlsReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newTlsSecrets")
	if err != nil {
		return res, err
	}
	expectedSecrets := d.([]corev1.Secret)

	for _, s := range expectedSecrets {
		if err = r.Client.Create(ctx, &s); err != nil {
			return res, errors.Wrapf(err, "Error when create secret %s", s.Name)
		}
	}

	return res, nil
}

// Update will update TLS secrets
func (r *TlsReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "tlsSecrets")
	if err != nil {
		return res, err
	}
	expectedSecrets := d.([]corev1.Secret)

	for _, s := range expectedSecrets {
		if err = r.Client.Update(ctx, &s); err != nil {
			return res, errors.Wrapf(err, "Error when update secret %s", s.Name)
		}
	}

	return res, nil
}

// Delete permit to delete TLS secrets
func (r *TlsReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldTlsSecrets")
	if err != nil {
		return res, err
	}
	oldSecrets := d.([]corev1.Secret)

	for _, s := range oldSecrets {
		if err = r.Client.Delete(ctx, &s); err != nil {
			return res, errors.Wrapf(err, "Error when delete secret %s", s.Name)
		}
	}

	return res, nil
}

// Diff permit to check if transport secrets are up to date
func (r *TlsReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var (
		d         any
		needRenew bool
		isUpdated bool
	)

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
	secretToUpdate := make([]corev1.Secret, 0)
	secretToCreate := make([]corev1.Secret, 0)
	secretToDelete := make([]corev1.Secret, 0)

	// phaseInit -> phaseCreate
	// Generate all certificates
	if sTransport == nil {
		r.Log.Debugf("Detect phase: %s", phaseCreate)

		diff.Diff.WriteString("Generate new certificates\n")

		// Generate transport PKI
		tmpTransportPki, transportRootCA, err := BuildTransportPkiSecret(o)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when generate transport PKI")
		}
		sTransportPki, isUpdated = updateSecret(sTransportPki, tmpTransportPki)
		if isUpdated {
			secretToUpdate = append(secretToUpdate, *sTransportPki)
		} else {
			err = ctrl.SetControllerReference(o, sTransportPki, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, *sTransportPki)
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
		secretToCreate = append(secretToCreate, *sTransport)

		// Handle API certificates
		if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {

			// Generate API PKI
			tmpApiPki, apiRootCA, err := BuildApiPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate API PKI")
			}
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if isUpdated {
				secretToUpdate = append(secretToUpdate, *sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, *sApiPki)
			}

			// Generate API certificate
			tmpApi, err := BuildApiSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate API certificate")
			}
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if isUpdated {
				secretToUpdate = append(secretToUpdate, *sApi)
			} else {
				err = ctrl.SetControllerReference(o, sApi, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, *sApi)
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

		data["newTlsSecrets"] = secretToCreate
		data["tlsSecrets"] = secretToUpdate
		data["oldTlsSecrets"] = secretToDelete
		data["phase"] = phaseCreate

		return diff, res, nil
	}

	// phaseGenerateTransportPki -> phasePropagateTransportPKI
	// Wait new CA propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateTransportPki, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateTransportPki, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phasePropagateTransportPki)
		data["phase"] = phasePropagateTransportPki
		return diff, res, nil
	}

	// phasePropagateTransportPki -> phaseUpdateCertificates
	// Generate all certificates
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateTransportPki, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate, metav1.ConditionFalse) {

		r.Log.Debugf("Detect phase: %s", phaseUpdateCertificates)

		// Generates certificates
		tmpTransportSecret, err := BuildTransportSecret(o, transportRootCA)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when renew transport certificates")
		}
		// Keep here the transitionnal CA appended in previous step
		tmpTransportSecret.Data["ca.crt"] = sTransport.Data["ca.crt"]
		diff.Diff.WriteString("Renew nodes certificates\n")
		sTransport, isUpdated = updateSecret(sTransport, tmpTransportSecret)
		if isUpdated {
			secretToUpdate = append(secretToUpdate, *sTransport)
		} else {
			err = ctrl.SetControllerReference(o, sTransport, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, *sTransport)
		}

		if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
			tmpApiPki, apiRootCA, err := BuildApiPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew API Pki")
			}
			diff.Diff.WriteString("Renew API Pki\n")
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if isUpdated {
				secretToUpdate = append(secretToUpdate, *sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, *sApiPki)
			}

			tmpApi, err := BuildApiSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew API certificate")
			}
			diff.Diff.WriteString("Renew API certificate\n")
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if isUpdated {
				secretToUpdate = append(secretToUpdate, *sApi)
			} else {
				err = ctrl.SetControllerReference(o, sApi, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, *sApi)
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

		data["newTlsSecrets"] = secretToCreate
		data["tlsSecrets"] = secretToUpdate
		data["oldTlsSecrets"] = secretToDelete
		data["phase"] = phaseUpdateCertificates

		return diff, res, nil
	}

	// phaseGenerateCertificate -> phasePropagateCertificate
	// Wait new certificates propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phasePropagateCertificates)

		data["phase"] = phasePropagateCertificates
		return diff, res, nil
	}

	// phaseCleanTransportCA -> phaseNormal
	// Remove old CA transport certificate
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsCondition, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseCleanTransportCA)

		if sTransport != nil && transportRootCA != nil {
			sTransport.Data["ca.crt"] = []byte(transportRootCA.GetCertificate())
			secretToUpdate = append(secretToUpdate, *sTransport)
			diff.NeedUpdate = true
			diff.Diff.WriteString(fmt.Sprintf("Clean old ca certificate from secret %s\n", sTransport.Name))
		}

		data["newTlsSecrets"] = secretToCreate
		data["tlsSecrets"] = secretToUpdate
		data["oldTlsSecrets"] = secretToDelete
		data["phase"] = phaseCleanTransportCA

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
			secretToCreate = append(secretToCreate, *sApiPki)
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
			secretToCreate = append(secretToCreate, *sApi)

		}
	}

	if !isRenew {
		// Check certificate validity if all certificates exists
		for name, crt := range certificates {
			needRenew, err = pki.NeedRenewCertificate(&crt, DefaultRenewCertificate, r.Log)
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
		r.Log.Debugf("Detect phase: %s", phaseUpdateTransportPki)

		// Renew only transport pki and wait all nodes get the new CA before to upgrade certificates
		tmpTransportPki, transportRootCA, err := BuildTransportPkiSecret(o)
		if err != nil {
			return diff, res, errors.Wrap(err, "Error when renew transport PKI")
		}
		diff.Diff.WriteString("Renew transport PKI\n")

		sTransportPki, isUpdated = updateSecret(sTransportPki, tmpTransportPki)
		if isUpdated {
			secretToUpdate = append(secretToUpdate, *sTransportPki)
		} else {
			err = ctrl.SetControllerReference(o, sTransportPki, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when set as owner reference")
			}
			secretToCreate = append(secretToCreate, *sTransportPki)
		}

		// Append new CA with others CA
		sTransport.Data["ca.crt"] = []byte(fmt.Sprintf("%s\n%s", string(sTransport.Data["ca.crt"]), transportRootCA.GetCertificate()))
		secretToUpdate = append(secretToUpdate, *sTransport)

		if len(secretToCreate) > 0 {
			diff.NeedCreate = true
		}
		if len(secretToUpdate) > 0 {
			diff.NeedUpdate = true
		}
		if len(secretToDelete) > 0 {
			diff.NeedDelete = true
		}

		data["newTlsSecrets"] = secretToCreate
		data["tlsSecrets"] = secretToUpdate
		data["oldTlsSecrets"] = secretToDelete
		data["phase"] = phaseUpdateTransportPki
		data["transportTlsSecret"] = sTransport

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
		secretToUpdate = append(secretToUpdate, *sTransport)
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []corev1.Secret{
		*sTransportPki,
	}
	if len(addedNode) == 0 && len(deletedNode) == 0 {
		// Not reconcile labels and annotation for transport secret if already updated on previous step
		secrets = append(secrets, *sTransport)
	}
	if o.IsTlsApiEnabled() && o.IsSelfManagedSecretForTlsApi() {
		secrets = append(secrets, *sApiPki, *sApi)
	}
	for _, s := range secrets {
		isUpdated := false
		if strDiff := common.DiffLabels(getLabels(o), s.Labels); strDiff != "" {
			diff.Diff.WriteString(strDiff + "\n")
			s.Labels = getLabels(o)
			isUpdated = true
		}
		if strDiff := common.DiffAnnotations(getAnnotations(o), s.Annotations); strDiff != "" {
			diff.Diff.WriteString(strDiff + "\n")
			s.Annotations = getAnnotations(o)
			isUpdated = true
		}

		if isUpdated {
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

	data["newTransportSecrets"] = secretToCreate
	data["tlsSecrets"] = secretToUpdate
	data["oldTlsSecrets"] = secretToDelete
	data["phase"] = phaseNormal

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *TlsReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

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
	o := resource.(*elasticsearchapi.Elasticsearch)
	var (
		d any
	)

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(tlsPhase)

	d, err = helper.Get(data, "transportTlsSecret")
	if err != nil {
		return res, err
	}
	sTransport := d.(*corev1.Secret)

	d, err = helper.Get(data, "transportPkiSecret")
	if err != nil {
		return res, err
	}
	sTransportPki := d.(*corev1.Secret)

	switch phase {
	case phaseCreate:

		// Check if pod already exist. Need to delete them to have a chance to reconcile
		// It's a kind of last hope
		// Need to delete all pod to force to reconcil with right ca / certificates
		podList := &corev1.PodList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchAnnotationKey))
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
			Message: "Tls ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateTransportPki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Transport PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateTransportPki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "CA transport certificate propagated",
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
	case phaseUpdateTransportPki:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateTransportPki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Transport PKI generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateTransportPki,
			Reason:  "Wait",
			Status:  metav1.ConditionFalse,
			Message: "Wait propagate new transport CA certificate",
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

		r.Log.Info("Phase to renew Transport PKI successfully finished")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	case phasePropagateTransportPki:
		// Get all statefullset and pods to check the new CA are successfully be propagated
		// Here, the ca.crt contain the old CA and the new CA
		// And Sts finished rolling upgrade
		stsList := &appv1.StatefulSetList{}
		podList := &corev1.PodList{}
		// Then, check if all Sts receive the new CA
		annotation := fmt.Sprintf("%s/ca-checksum", elasticsearchAnnotationKey)
		caChecksum := fmt.Sprintf("%x", sha256.Sum256(sTransportPki.Data["ca.crt"]))
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
		}

		// First, check all pod ready on Satefulset already upgraded
		for _, sts := range stsList.Items {
			if sts.Spec.Template.Annotations[annotation] != "" && sts.Spec.Template.Annotations[annotation] == caChecksum {
				labelSelectors = labels.SelectorFromSet(sts.Spec.Template.Labels)
				if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
					return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
				}

				isFinished := true
				for _, p := range podList.Items {
					// The pod must have CA checksum annotation and need to be ready
					if p.Annotations[annotation] == "" || p.Annotations[annotation] != caChecksum || !podutil.IsPodReady(&p) {
						isFinished = false
					}
				}
				if !isFinished {
					// All Sts not yet finished upgrade
					r.Log.Info("Phase propagate transport CA: wait pod to be ready")
					return ctrl.Result{RequeueAfter: time.Second * 30}, nil
				}
			}
		}

		for _, sts := range stsList.Items {

			// Check CA is updated
			if sts.Spec.Template.Annotations[annotation] == "" || sts.Spec.Template.Annotations[annotation] != caChecksum {
				sts.Spec.Template.Annotations[annotation] = caChecksum
				if err = r.Client.Update(ctx, &sts); err != nil {
					return res, errors.Wrapf(err, "Error when update statefullset %s", sts.Name)
				}

				condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
					Type:    TlsConditionPropagateTransportPki,
					Reason:  "Propagate",
					Status:  metav1.ConditionFalse,
					Message: fmt.Sprintf("Propagate transport PKI on %s", sts.Name),
				})

				r.Log.Infof("Phase propagate transport CA: wait statefulset %s start with new CA", sts.Name)

				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

			}
		}

		// all CA upgrade are finished
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateTransportPki,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Transport PKI generated",
		})

		r.Log.Info("Phase propagate transport CA: all statefulset restarted successfully with new CA")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil

	case phaseUpdateCertificates:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		r.Log.Info("Phase propagate certificates: all certificates have been successfully renewed")
		return ctrl.Result{Requeue: true}, nil

	case phasePropagateCertificates:
		// Get all statefullset to check the new certificates to be propagated
		// Here, ca.crt contain only the new CA
		// And Sts finished rolling upgrade
		stsList := &appv1.StatefulSetList{}
		podList := &corev1.PodList{}
		annotation := fmt.Sprintf("%s/ca-checksum", elasticsearchAnnotationKey)
		caChecksum := fmt.Sprintf("%x", sha256.Sum256(sTransport.Data["ca.crt"]))
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
		}
		if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
		}

		// First, check if one sts currently upgraded
		for _, sts := range stsList.Items {
			if sts.Spec.Template.Annotations[annotation] != "" && sts.Spec.Template.Annotations[annotation] == caChecksum {
				labelSelectors = labels.SelectorFromSet(sts.Spec.Template.Labels)
				if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
					return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
				}

				isFinished := true
				for _, p := range podList.Items {
					// The pod must have CA checksum annotation and need to be ready
					if p.Annotations[annotation] == "" || p.Annotations[annotation] != caChecksum || !podutil.IsPodReady(&p) {
						isFinished = false
					}
				}
				if !isFinished {
					// All Sts not yet finished upgrade
					r.Log.Info("Phase propagate certificates: wait pod to be ready")
					return ctrl.Result{RequeueAfter: time.Second * 30}, nil
				}
			}
		}

		// Then, check if all Sts receive the new certificates
		for _, sts := range stsList.Items {
			// Check CA is updated
			if sts.Spec.Template.Annotations[annotation] == "" || sts.Spec.Template.Annotations[annotation] != caChecksum {
				sts.Spec.Template.Annotations[annotation] = caChecksum
				if err = r.Client.Update(ctx, &sts); err != nil {
					return res, errors.Wrapf(err, "Error when update statefullset %s", sts.Name)
				}

				condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
					Type:    TlsConditionPropagateCertificate,
					Reason:  "Propagate",
					Status:  metav1.ConditionFalse,
					Message: fmt.Sprintf("Propagate certificate on %s", sts.Name),
				})

				r.Log.Infof("Phase propagate certificates: wait statefulset %s restart with new certificates", sts.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}

		// all certificate upgrade are finished
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates propagated",
		})

		r.Log.Info("Phase propagate certificates: all nodes have beed successfully restarted with new certificates")

		return ctrl.Result{Requeue: true}, nil

	case phaseCleanTransportCA:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Tls ready",
		})

		r.Log.Info("Clean old transport CA certificate successfully")

	}

	return res, nil
}
