package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	secretElasticsearchCondition shared.ConditionName = "SecretElasticsearchReady"
	secretElasticsearchPhase     shared.PhaseName     = "SecretElasticsearch"
)

type secretElasticsearchReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Elasticsearch, *corev1.Secret]
}

func newSecretElasticsearchReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Elasticsearch, *corev1.Secret]) {
	return &secretElasticsearchReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*discovercrd.Elasticsearch, *corev1.Secret](
			client,
			secretElasticsearchPhase,
			secretElasticsearchCondition,
			recorder,
			common.FieldManager,
		),
	}
}

// Read existing Elasticsearch secret
func (r *secretElasticsearchReconciler) Read(ctx context.Context, o *discovercrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
	secretList := &corev1.SecretList{}
	var (
		elasticsearchCluster        *elasticsearchcrd.Elasticsearch
		elasticsearchCaSecret       *corev1.Secret
		elasticsearchUserSecret     *corev1.Secret
		elasticsearchCustomCaSecret *corev1.Secret
	)

	// Read existing secrets
	labelSelectors, err := labels.Parse(fmt.Sprintf("discoverName=%s,%s=true", o.Name, discovercrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, secretList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read secrets")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(secretList.Items))

	// Read Managed Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {

		// Read elasticsearch cluster
		elasticsearchCluster = &elasticsearchcrd.Elasticsearch{}
		namespace := o.Namespace
		if o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
			namespace = o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
		}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: namespace, Name: o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}, elasticsearchCluster); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read elasticsearch %s", o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)
			}
			logger.Warnf("Elasticsearch %s not yet exist, try again later", o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Get the CA secret
		if elasticsearchCluster.Spec.Tls.IsTlsEnabled() && elasticsearchCluster.Spec.Tls.IsSelfManagedSecretForTls() {
			elasticsearchCaSecret = &corev1.Secret{}
			name := elasticsearchcontrollers.GetSecretNameForTlsApi(elasticsearchCluster)
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: elasticsearchCluster.Namespace, Name: name}, elasticsearchCaSecret); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", name)
				}
				logger.Warnf("Secret not found %s/%s, try latter", elasticsearchCluster.Namespace, name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			data["elasticsearchCASecretRef"] = ptr.To(elasticsearchCaSecret.Name)
		}
	}

	// Read credentials secret (both managed and external)
	if o.Spec.ElasticsearchRef.SecretRef != nil && o.Spec.ElasticsearchRef.SecretRef.Name != "" {
		elasticsearchUserSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.SecretRef.Name}, elasticsearchUserSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.SecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.ElasticsearchRef.SecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		data["elasticsearchUserSecretRef"] = ptr.To(elasticsearchUserSecret.Name)
	}

	// Read custom CA secret
	if o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil && o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name != "" {
		elasticsearchCustomCaSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}, elasticsearchCustomCaSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Build expected secret
	expectedSecretsElasticsearch, err := buildElasticsearchSecrets(o, elasticsearchCluster, elasticsearchCaSecret, elasticsearchUserSecret, elasticsearchCustomCaSecret)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when build expected secret")
	}
	read.SetExpectedObjects(expectedSecretsElasticsearch)

	return read, res, nil
}

// Diff permit to check if Elasticsearch secrets are up to date
func (r *secretElasticsearchReconciler) Diff(ctx context.Context, o *discovercrd.Elasticsearch, read multiphase.MultiPhaseRead[*corev1.Secret], data map[string]any, logger *logrus.Entry) (diff multiphase.MultiPhaseDiff[*corev1.Secret], res reconcile.Result, err error) {
	diff = multiphase.NewMultiPhaseDiff[*corev1.Secret]()
	tmpCurrentObjects := make([]*corev1.Secret, len(read.GetCurrentObjects()))
	copy(tmpCurrentObjects, read.GetCurrentObjects())

	for _, expected := range read.GetExpectedObjects() {
		isFound := false

		// Set ownerReferences on expected object before to diff them
		err = ctrl.SetControllerReference(o, expected, r.Client().Scheme())
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", expected.GetName())
		}

		for i, current := range tmpCurrentObjects {
			if expected.Name == current.Name {
				isFound = true

				// Calculate diff
				patchResult, err := patch.DefaultPatchMaker.Calculate(current, expected)
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when calculate diff for secret %s/%s", current.Namespace, current.Name)
				}
				if !patchResult.IsEmpty() {
					updatedObject := patchResult.Patched.(*corev1.Secret)
					updatedObject.Data = expected.Data
					diff.AddDiff(fmt.Sprintf("diff %s: %s", updatedObject.GetName(), string(patchResult.Patch)))
					diff.AddObjectToUpdate(updatedObject)
					logger.Debugf("Need update object '%s'", updatedObject.GetName())
				}

				// Remove items found
				tmpCurrentObjects = helper.DeleteItemFromSlice(tmpCurrentObjects, i)

				break
			}
		}

		if !isFound {
			// Need create object
			diff.AddDiff(fmt.Sprintf("Need Create object '%s'", expected.GetName()))
			diff.AddObjectToCreate(expected)

			logger.Debugf("Need create object '%s'", expected.GetName())
		}
	}

	// Need delete
	if len(tmpCurrentObjects) > 0 {
		diff.SetObjectsToDelete(tmpCurrentObjects)
		for _, object := range tmpCurrentObjects {
			diff.AddDiff(fmt.Sprintf("Need delete object '%s'", object.GetName()))
		}
	}

	return diff, res, nil
}
