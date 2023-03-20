package metricbeat

import (
	"context"
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
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
	CredentialCondition = "CredentialReady"
	CredentialPhase     = "Credential"
)

type CredentialReconciler struct {
	common.Reconciler
}

func NewCredentialReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &CredentialReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "credential",
			}),
			Name:   "credential",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *CredentialReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, CredentialCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   CredentialCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = CredentialPhase

	return res, nil
}

// Read existing secret
func (r *CredentialReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)
	s := &corev1.Secret{}
	sEs := &corev1.Secret{}

	var es *elasticsearchcrd.Elasticsearch

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
		}
		s = nil
	}
	data["currentObject"] = s

	if o.Spec.ElasticsearchRef.IsManaged() {

		// Read Elasticsearch
		es, err = common.GetElasticsearchFromRef(ctx, r.Client, o, o.Spec.ElasticsearchRef)
		if err != nil {
			return res, errors.Wrap(err, "Error when read elasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Read secret that store Elasticsearch crdentials
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForCredentials(es)}, sEs); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForCredentials(es))
			}
			r.Log.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForCredentials(es))
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Generate expected secret
	expectedSecretCredential, err := BuildCredentialSecret(o, sEs)
	if err != nil {
		return res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCredentials(o))
	}
	data["expectedObject"] = expectedSecretCredential

	return res, nil
}

// Diff permit to check if credential secret is up to date
func (r *CredentialReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *CredentialReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    CredentialCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *CredentialReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Credential secret successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, CredentialCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    CredentialCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
