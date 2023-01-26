package kibana

import (
	"context"
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	appv1 "k8s.io/api/apps/v1"
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
	DeploymentCondition = "DeploymentReady"
	DeploymentPhase     = "Deployment"
)

type DeploymentReconciler struct {
	common.Reconciler
}

func NewDeploymentReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &DeploymentReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "deployment",
			}),
			Name:   "deployment",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *DeploymentReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, DeploymentCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   DeploymentCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = DeploymentPhase
	}

	return res, nil
}

// Read existing satefulsets
func (r *DeploymentReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	dpl := &appv1.Deployment{}
	secretKeystore := &corev1.Secret{}
	secretApiCrt := &corev1.Secret{}
	secretCustomCAElasticsearch := &corev1.Secret{}
	var es *elasticsearchcrd.Elasticsearch

	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetDeploymentName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read deployment")
		}
		dpl = nil
	}

	data["currentObject"] = dpl

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {
		es, err = GetElasticsearchRef(ctx, r.Client, o)
		if err != nil {
			return res, errors.Wrap(err, "Error when read ElasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		es = nil
	}

	// Read keystore secret if needed
	if o.Spec.KeystoreSecretRef != nil && o.Spec.KeystoreSecretRef.Name != "" {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.KeystoreSecretRef.Name}, secretKeystore); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.KeystoreSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.KeystoreSecretRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		secretKeystore = nil
	}

	// Read APi Crt if needed
	if o.IsTlsEnabled() {
		if o.IsSelfManagedSecretForTls() {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTls(o)}, secretApiCrt); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTls(o))
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTls(o))
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		} else {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, secretApiCrt); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	} else {
		secretApiCrt = nil
	}

	// Read Custom CA Elasticsearch if needed
	if (o.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled()) || o.Spec.Tls.ElasticsearchCaSecretRef != nil {
		if o.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, secretCustomCAElasticsearch); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForCAElasticsearch(o))
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		} else {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.ElasticsearchCaSecretRef.Name}, secretCustomCAElasticsearch); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.ElasticsearchCaSecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.ElasticsearchCaSecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			if len(secretCustomCAElasticsearch.Data["ca.crt"]) == 0 {
				return res, errors.Errorf("Secret %s must have a key `ca.crt`", secretCustomCAElasticsearch.Name)
			}
		}
	} else {
		secretCustomCAElasticsearch = nil
	}

	// Generate expected deployment
	expectedDeployment, err := BuildDeployment(o, es, secretKeystore, secretApiCrt, secretCustomCAElasticsearch)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate deployment")
	}
	data["expectedObject"] = expectedDeployment

	return res, nil
}

// Diff permit to check if deployment is up to date
func (r *DeploymentReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *DeploymentReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    DeploymentCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *DeploymentReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Deployment successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, DeploymentCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    DeploymentCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
