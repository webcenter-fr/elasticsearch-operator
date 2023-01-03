package kibana

import (
	"context"
	"fmt"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CAElasticsearchCondition = "CAElasticsearchReady"
	CAElasticsearchPhase     = "CAElasticsearch"
)

type CAElasticsearchReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewCAElasticsearchReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &CAElasticsearchReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "caElasticsearch",
	}
}

// Name return the current phase
func (r *CAElasticsearchReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *CAElasticsearchReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, CAElasticsearchCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   CAElasticsearchCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = CAElasticsearchPhase
	}

	return res, nil
}

// Read existing secret
func (r *CAElasticsearchReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {

	o := resource.(*kibanacrd.Kibana)
	s := &corev1.Secret{}
	sEs := &corev1.Secret{}

	var es *elasticsearchcrd.Elasticsearch
	var expectedSecretCAElasticsearch *corev1.Secret

	// Check if mirror CAApiPKI from Elasticsearch CRD
	if !o.IsElasticsearchRef() {
		return res, nil
	}

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
		}
		s = nil
	}
	data["currentSecretCAElasticsearch"] = s

	// Read Elasticsearch
	es, err = GetElasticsearchRef(ctx, r.Client, o)
	if err != nil {
		return res, errors.Wrap(err, "Error when read elasticsearchRef")
	}
	if es == nil {
		r.Log.Warn("ElasticsearchRef not found, try latter")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Check if mirror CAApiPKI from Elasticsearch CRD
	if !es.IsTlsApiEnabled() {
		data["expectedSecretCAElasticsearch"] = expectedSecretCAElasticsearch
		return res, nil
	}

	// Read secret that store Elasticsearch ApiPKI
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForPkiApi(es)}, sEs); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForPkiApi(es))
		}
		r.Log.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForPkiApi(es))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Generate expected secret
	expectedSecretCAElasticsearch, err = BuildCAElasticsearchSecret(o, sEs)
	if err != nil {
		return res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCAElasticsearch(o))
	}
	data["expectedSecretCAElasticsearch"] = expectedSecretCAElasticsearch

	return res, nil
}

// Create will create secret
func (r *CAElasticsearchReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newSecrets")
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

// Update will update secret
func (r *CAElasticsearchReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "secrets")
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

// Delete permit to delete secret
func (r *CAElasticsearchReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldSecrets")
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

// Diff permit to check if credential secret is up to date
func (r *CAElasticsearchReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	var d any

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	// No PkiCA to mirror
	if !o.IsElasticsearchRef() {
		return diff, res, nil
	}

	d, err = helper.Get(data, "currentSecretCAElasticsearch")
	if err != nil {
		return diff, res, err
	}
	currentSecretCAElasticsearch := d.(*corev1.Secret)

	d, err = helper.Get(data, "expectedSecretCAElasticsearch")
	if err != nil {
		return diff, res, err
	}
	expectedSecretCAElasticsearch := d.(*corev1.Secret)

	secretToUpdate := make([]corev1.Secret, 0)
	secretToCreate := make([]corev1.Secret, 0)
	secretToDelete := make([]corev1.Secret, 0)

	// No PkiCA to mirror
	if expectedSecretCAElasticsearch == nil {
		return diff, res, nil
	}

	if currentSecretCAElasticsearch == nil {

		// Create new credential secret
		diff.NeedCreate = true
		diff.Diff.WriteString(fmt.Sprintf("Create secret %s\n", expectedSecretCAElasticsearch.Name))

		// Set owner
		err = ctrl.SetControllerReference(o, expectedSecretCAElasticsearch, r.Scheme)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference")
		}

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedSecretCAElasticsearch); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on secret %s", expectedSecretCAElasticsearch.Name)
		}

		secretToCreate = append(secretToCreate, *expectedSecretCAElasticsearch)

	} else {

		// Check if need update secret
		patchResult, err := patch.DefaultPatchMaker.Calculate(currentSecretCAElasticsearch, expectedSecretCAElasticsearch, patch.CleanMetadata(), patch.IgnoreStatusFields())
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when diffing secret %s", currentSecretCAElasticsearch.Name)
		}
		if !patchResult.IsEmpty() {
			updatedSecret := patchResult.Patched.(*corev1.Secret)
			diff.NeedUpdate = true
			diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedSecret.Name, string(patchResult.Patch)))
			secretToUpdate = append(secretToUpdate, *updatedSecret)
			r.Log.Debugf("Need update secret %s", updatedSecret.Name)
		}
	}

	data["newSecrets"] = secretToCreate
	data["secrets"] = secretToUpdate
	data["oldSecrets"] = secretToDelete

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *CAElasticsearchReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    CAElasticsearchCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *CAElasticsearchReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "CA Elasticsearch secret successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, CAElasticsearchCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    CAElasticsearchCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
