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
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type statefulsetPhase string

const (
	DeploymentCondition = "DeploymentReady"
	DeploymentPhase     = "Deployment"
)

type DeploymentReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewDeploymentReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &DeploymentReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "deployment",
	}
}

// Name return the current phase
func (r *DeploymentReconciler) Name() string {
	return r.name
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

	data["currentDeployment"] = dpl

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
	if o.IsTlsEnabled() && !o.IsSelfManagedSecretForTls() {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, secretApiCrt); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		secretApiCrt = nil
	}

	// Read Custom CA Elasticsearch if needed
	if !o.Spec.ElasticsearchRef.IsManaged() && o.Spec.Tls.ElasticsearchCaSecretRef != nil {
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
	} else {
		secretCustomCAElasticsearch = nil
	}

	// Generate expected deployment
	expectedDeployment, err := BuildDeployment(o, es, secretKeystore, secretApiCrt, secretCustomCAElasticsearch)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate deployment")
	}
	data["expectedDeployment"] = expectedDeployment

	return res, nil
}

// Create will create deployment
func (r *DeploymentReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newDeployments")
	if err != nil {
		return res, err
	}
	expectedDpls := d.([]appv1.Deployment)

	for _, dpl := range expectedDpls {
		if err = r.Client.Create(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when create deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Update will update deployment
func (r *DeploymentReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "deployments")
	if err != nil {
		return res, err
	}
	expectedDpls := d.([]appv1.Deployment)

	for _, dpl := range expectedDpls {
		if err = r.Client.Update(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when update deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Delete permit to delete deployment
func (r *DeploymentReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldDeployments")
	if err != nil {
		return res, err
	}
	oldDpls := d.([]appv1.Deployment)

	for _, dpl := range oldDpls {
		if err = r.Client.Delete(ctx, &dpl); err != nil {
			return res, errors.Wrapf(err, "Error when delete deployment %s", dpl.Name)
		}
	}

	return res, nil
}

// Diff permit to check if deployment is up to date
func (r *DeploymentReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	var d any

	d, err = helper.Get(data, "currentDeployment")
	if err != nil {
		return diff, res, err
	}
	currentDeployment := d.(*appv1.Deployment)

	d, err = helper.Get(data, "expectedDeployment")
	if err != nil {
		return diff, res, err
	}
	expectedDeployment := d.(*appv1.Deployment)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	dplToUpdate := make([]appv1.Deployment, 0)
	dplToCreate := make([]appv1.Deployment, 0)
	dplToDelete := make([]appv1.Deployment, 0)

	// Deployment not yet exist
	if currentDeployment == nil {
		diff.NeedCreate = true
		diff.Diff.WriteString(fmt.Sprintf("Deployment %s not yet exist\n", expectedDeployment.Name))

		// Set owner
		err = ctrl.SetControllerReference(o, expectedDeployment, r.Scheme)
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference")
		}

		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedDeployment); err != nil {
			return diff, res, errors.Wrapf(err, "Error when set diff annotation on deployment %s", expectedDeployment.Name)
		}

		dplToCreate = append(dplToCreate, *expectedDeployment)

		r.Log.Debugf("Need create deployment %s", expectedDeployment.Name)

		data["newDeployments"] = dplToCreate
		data["deployments"] = dplToUpdate
		data["oldDeployment"] = dplToDelete

		return diff, res, nil
	}

	// Check if deployment is up to date
	patchResult, err := patch.DefaultPatchMaker.Calculate(currentDeployment, expectedDeployment, patch.CleanMetadata(), patch.IgnoreStatusFields())
	if err != nil {
		return diff, res, errors.Wrapf(err, "Error when diffing deployment %s", currentDeployment.Name)
	}
	if !patchResult.IsEmpty() {
		updatedDpl := patchResult.Patched.(*appv1.Deployment)
		diff.NeedUpdate = true
		diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedDpl.Name, string(patchResult.Patch)))

		dplToUpdate = append(dplToUpdate, *updatedDpl)
		r.Log.Debugf("Need update deployment %s", updatedDpl.Name)
	}

	data["newDeployments"] = dplToCreate
	data["deployments"] = dplToUpdate
	data["oldDeployment"] = dplToDelete

	return diff, res, nil

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
