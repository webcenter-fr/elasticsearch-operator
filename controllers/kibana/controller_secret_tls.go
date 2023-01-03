package kibana

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/codingsince1985/checksum"
	"github.com/disaster37/goca"
	"github.com/disaster37/goca/cert"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
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
	TlsConditionGenerateCertificate  = "TlsGenerateCertificates"
	TlsConditionPropagateCertificate = "TlsPropagateCertificates"
	TlsCondition                     = "TlsReady"
	TlsPhase                         = "Tls"
	DefaultRenewCertificate          = -time.Hour * 24 * 30 // 30 days before expired
)

var (
	phaseTlsCreate                tlsPhase = "tlsCreate"
	phaseTlsUpdateCertificates    tlsPhase = "tlsUpdateCertificates"
	phaseTlsPropagateCertificates tlsPhase = "tlsPropagateCertificates"
	phaseTlsNormal                tlsPhase = "tlsNormal"
)

type TlsReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewTlsReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
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
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, TlsCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = TlsPhase
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
	o := resource.(*kibanacrd.Kibana)
	sApi := &corev1.Secret{}
	sApiPki := &corev1.Secret{}
	var (
		apiRootCA  *goca.CA
		apiCrt     *x509.Certificate
		secretName string
	)

	if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {
		// Read API PKI secret
		secretName = GetSecretNameForPki(o)
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApiPki); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApiPki = nil
		}

		// Read API secret
		secretName = GetSecretNameForTls(o)
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: secretName}, sApi); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read existing secret %s", secretName)
			}
			sApi = nil
		}
	}

	// Load API PKI
	if sApiPki != nil {
		// Load root CA
		apiRootCA, err = pki.LoadRootCATransport(sApiPki.Data["ca.key"], sApiPki.Data["ca.pub"], sApiPki.Data["ca.crt"], sApiPki.Data["ca.crl"], r.Log)
		if err != nil {
			return res, errors.Wrap(err, "Error when load PKI")
		}
	}

	// Load API certificate
	if sApi != nil {
		apiCrt, err = cert.LoadCertFromPem(sApi.Data["tls.crt"])
		if err != nil {
			return res, errors.Wrapf(err, "Error when load certificate")
		}
	}

	data["apiRootCA"] = apiRootCA
	data["apiCertificate"] = apiCrt
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
	o := resource.(*kibanacrd.Kibana)
	var (
		d         any
		needRenew bool
		isUpdated bool
	)

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
	if sApi == nil || sApiPki == nil {
		r.Log.Debugf("Detect phase: %s", phaseTlsCreate)

		diff.Diff.WriteString("Generate new certificates\n")

		// Handle API certificates
		if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {

			// Generate API PKI
			tmpApiPki, apiRootCA, err := BuildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate PKI")
			}
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
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
			tmpApi, err := BuildTlsSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when generate certificate")
			}
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
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
		data["phase"] = phaseTlsCreate

		return diff, res, nil
	}

	// phaseGenerateCertificate -> phasePropagateCertificate
	// Wait new certificates propagated on all Elasticsearch instance
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionGenerateCertificate, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionPropagateCertificate, metav1.ConditionFalse) {
		r.Log.Debugf("Detect phase: %s", phaseTlsPropagateCertificates)

		data["phase"] = phaseTlsPropagateCertificates
		return diff, res, nil
	}

	// Check if certificates will expire
	isRenew := false
	certificates := map[string]x509.Certificate{}
	if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {
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
		r.Log.Debugf("Detect phase: %s", phaseTlsUpdateCertificates)

		if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {
			tmpApiPki, apiRootCA, err := BuildPkiSecret(o)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew Pki")
			}
			diff.Diff.WriteString("Renew API Pki\n")
			sApiPki, isUpdated = updateSecret(sApiPki, tmpApiPki)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApiPki); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApiPki.Name)
			}
			if isUpdated {
				secretToUpdate = append(secretToUpdate, *sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, *sApiPki)
			}

			tmpApi, err := BuildTlsSecret(o, apiRootCA)
			if err != nil {
				return diff, res, errors.Wrap(err, "Error when renew certificate")
			}
			diff.Diff.WriteString("Renew API certificate\n")
			sApi, isUpdated = updateSecret(sApi, tmpApi)
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sApi); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", sApi.Name)
			}
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
		data["phase"] = phaseTlsUpdateCertificates

		return diff, res, nil
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []corev1.Secret{}
	if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {
		secrets = append(secrets, *sApiPki, *sApi)
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
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&s); err != nil {
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

	data["newTransportSecrets"] = secretToCreate
	data["tlsSecrets"] = secretToUpdate
	data["oldTlsSecrets"] = secretToDelete
	data["phase"] = phaseTlsNormal

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *TlsReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

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
	o := resource.(*kibanacrd.Kibana)
	var (
		d any
	)

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(tlsPhase)

	d, err = helper.Get(data, "apiTlsSecret")
	if err != nil {
		return res, err
	}
	sApi := d.(*corev1.Secret)

	switch phase {
	case phaseTlsCreate:
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Tls secrets successfully generated")

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
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

	case phaseTlsUpdateCertificates:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "NotReady",
			Status:  metav1.ConditionFalse,
			Message: "Generete new certificates",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionGenerateCertificate,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Certificates generated",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsConditionPropagateCertificate,
			Reason:  "NotPropaged",
			Status:  metav1.ConditionFalse,
			Message: "Not yet propaged",
		})

		r.Log.Info("Phase propagate certificates: all certificates have been successfully renewed")
		return ctrl.Result{Requeue: true}, nil

	case phaseTlsPropagateCertificates:
		// Get deployment to check the new certificates to be propagated
		// And deployment finished rolling upgrade

		dpl := &appv1.Deployment{}
		podList := &corev1.PodList{}
		annotation := fmt.Sprintf("%s/ca-checksum", KibanaAnnotationKey)
		caChecksum, err := checksum.SHA256sumReader(bytes.NewReader(sApi.Data["ca.crt"]))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate checksum from ca.crt")
		}
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Name}, dpl); err != nil {
			if k8serrors.IsNotFound(err) {
				condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
					Type:    TlsConditionPropagateCertificate,
					Reason:  "Ready",
					Status:  metav1.ConditionTrue,
					Message: "Certificates propaged",
				})
				condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
					Type:    TlsCondition,
					Reason:  "Ready",
					Status:  metav1.ConditionTrue,
					Message: "Generete new certificates",
				})

				r.Log.Info("Phase propagate certificates: all certificates have been successfully renewed")
				return ctrl.Result{Requeue: true}, nil
			}
			return res, errors.Wrapf(err, "Error when read Elasticsearch statefullsets")
		}

		// First, check if deployment currently upgraded
		if dpl.Spec.Template.Annotations[annotation] != "" && dpl.Spec.Template.Annotations[annotation] == caChecksum {
			labelSelectors := labels.SelectorFromSet(dpl.Spec.Template.Labels)
			if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
				return res, errors.Wrapf(err, "Error when read Kibana pods")
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

		// Then, check if deployment receive the new certificates
		// Check CA is updated
		if dpl.Spec.Template.Annotations[annotation] == "" || dpl.Spec.Template.Annotations[annotation] != caChecksum {
			dpl.Spec.Template.Annotations[annotation] = caChecksum
			if err = r.Client.Update(ctx, dpl); err != nil {
				return res, errors.Wrapf(err, "Error when update deployment %s", dpl.Name)
			}

			condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:    TlsConditionPropagateCertificate,
				Reason:  "Propagate",
				Status:  metav1.ConditionFalse,
				Message: "Propagate certificate",
			})

			r.Log.Infof("Phase propagate certificates: wait deployment %s restart with new certificates", dpl.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
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
	}

	return res, nil
}
