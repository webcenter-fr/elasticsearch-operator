package kibana

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/disaster37/goca"
	"github.com/disaster37/goca/cert"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/pki"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TlsCondition            = "TlsReady"
	TlsPhase                = "Tls"
	DefaultRenewCertificate = -time.Hour * 24 * 30 // 30 days before expired
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
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, TlsCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   TlsCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = TlsPhase

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

// Diff permit to check if transport secrets are up to date
func (r *TlsReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	var (
		d         any
		needRenew bool
		isUpdated bool
	)

	defaultRenewCertificate := DefaultRenewCertificate
	if o.Spec.Tls.RenewalDays != nil {
		defaultRenewCertificate = time.Duration(*o.Spec.Tls.RenewalDays) * 24 * time.Hour
	}

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
	secretToUpdate := make([]client.Object, 0)
	secretToCreate := make([]client.Object, 0)
	secretToDelete := make([]client.Object, 0)

	// Generate all certificates
	if sApi == nil || sApiPki == nil {
		r.Log.Debugf("Generate all certificates")

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
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApiPki)
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
		r.Log.Debugf("Renew all certificates")

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
				secretToUpdate = append(secretToUpdate, sApiPki)
			} else {
				err = ctrl.SetControllerReference(o, sApiPki, r.Scheme)
				if err != nil {
					return diff, res, errors.Wrap(err, "Error when set as owner reference")
				}
				secretToCreate = append(secretToCreate, sApiPki)
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

		return diff, res, nil
	}

	// Check if labels or annotations need to bu upgraded
	secrets := []*corev1.Secret{}
	if o.IsTlsEnabled() && o.IsSelfManagedSecretForTls() {
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

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "TLS certificates successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    TlsCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
