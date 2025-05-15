package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"time"

	"emperror.dev/errors"
	elasticsearchhandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
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
	StatefulsetCondition            shared.ConditionName = "StatefulsetReady"
	StatefulsetConditionUpgrade     shared.ConditionName = "StatefulsetUpgrade"
	StatefulsetPhase                shared.PhaseName     = "Statefullset"
	StatefulsetPhaseUpgradeStarted  shared.PhaseName     = "statefulsetUpgradeStarted"
	StatefulsetPhaseUpgrade         shared.PhaseName     = "statefulsetUpgrade"
	StatefulsetPhaseUpgradeFinished shared.PhaseName     = "statefulsetUpgradeFinished"
	StatefulsetPhaseNormal          shared.PhaseName     = "statefulsetNormal"
)

type statefulsetReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet]
	isOpenshift bool
}

func newStatefulsetReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet]) {
	return &statefulsetReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet](
			client,
			StatefulsetPhase,
			StatefulsetCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing satefulsets
func (r *statefulsetReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*appv1.StatefulSet], res reconcile.Result, err error) {
	stsList := &appv1.StatefulSetList{}
	read = multiphase.NewMultiPhaseRead[*appv1.StatefulSet]()
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	configMapsChecksum := make([]*corev1.ConfigMap, 0)
	secretsChecksum := make([]*corev1.Secret, 0)

	// Read current satefulsets
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read statefulset")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(stsList.Items))

	// Read keystore secret if needed
	if o.Spec.GlobalNodeGroup.KeystoreSecretRef != nil && o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name != "" {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Read cacerts secret if needed
	if o.Spec.GlobalNodeGroup.CacertsSecretRef != nil && o.Spec.GlobalNodeGroup.CacertsSecretRef.Name != "" {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.GlobalNodeGroup.CacertsSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.GlobalNodeGroup.CacertsSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.GlobalNodeGroup.CacertsSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Read API certificate secret if needed
	if o.Spec.Tls.IsTlsEnabled() {
		if o.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsApi(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsApi(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsApi(o))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
		} else {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
		}
	}

	// Read transport certicate secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsTransport(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsTransport(o))
		}
		logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsTransport(o))
		return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}
	secretsChecksum = append(secretsChecksum, s)

	// Read configMaps to generate checksum
	// Keep only configmap of type config
	labelSelectors, err = labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	for _, cm := range cmList.Items {
		if cm.Annotations[fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey)] == "config" {
			configMapsChecksum = append(configMapsChecksum, &cm)
		}
	}

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.GlobalNodeGroup.AdditionalVolumes {
		if v.ConfigMap != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.ConfigMap.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", v.ConfigMap.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", v.ConfigMap.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, cm)
			break
		}

		if v.Secret != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Secret.SecretName}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", v.Secret.SecretName)
				}
				logger.Warnf("Secret %s not yet exist, try again later", v.Secret.SecretName)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
			break
		}
	}

	envList := make([]corev1.EnvVar, 0, len(o.Spec.GlobalNodeGroup.Env))
	envFromList := make([]corev1.EnvFromSource, 0, len(o.Spec.GlobalNodeGroup.EnvFrom))

	// Compute all env and envFrom
	envList = append(envList, o.Spec.GlobalNodeGroup.Env...)
	envFromList = append(envFromList, o.Spec.GlobalNodeGroup.EnvFrom...)
	for _, nodeGroup := range o.Spec.NodeGroups {
		envList = append(envList, nodeGroup.Env...)
		envFromList = append(envFromList, nodeGroup.EnvFrom...)
	}

	// Read extra Env to generate checksum if secret or configmap
	for _, env := range envList {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.SecretKeyRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", env.ValueFrom.SecretKeyRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", env.ValueFrom.SecretKeyRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
			break
		}

		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", env.ValueFrom.ConfigMapKeyRef.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", env.ValueFrom.ConfigMapKeyRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, cm)
			break
		}
	}

	// Read extra Env from to generate checksum if secret or configmap
	for _, ef := range envFromList {
		if ef.SecretRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.SecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", ef.SecretRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", ef.SecretRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
			break
		}

		if ef.ConfigMapRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.ConfigMapRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", ef.ConfigMapRef.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", ef.ConfigMapRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, cm)
			break
		}
	}

	// Generate expected statefulsets
	expectedSts, err := buildStatefulsets(o, secretsChecksum, configMapsChecksum, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate statefulsets")
	}
	read.SetExpectedObjects(expectedSts)

	return read, res, nil
}

// Diff permit to check if statefulset is up to date
func (r *statefulsetReconciler) Diff(ctx context.Context, o *elasticsearchcrd.Elasticsearch, read multiphase.MultiPhaseRead[*appv1.StatefulSet], data map[string]any, logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff multiphase.MultiPhaseDiff[*appv1.StatefulSet], res reconcile.Result, err error) {
	var esHandler elasticsearchhandler.ElasticsearchHandler
	var v any

	currentStatefulsets := read.GetCurrentObjects()
	copyCurrentStatefulsets := make([]*appv1.StatefulSet, len(currentStatefulsets))
	copy(copyCurrentStatefulsets, currentStatefulsets)
	expectedStatefulsets := read.GetExpectedObjects()

	diff = multiphase.NewMultiPhaseDiff[*appv1.StatefulSet]()
	stsToExpectedUpdated := make([]*appv1.StatefulSet, 0)
	data["phase"] = StatefulsetPhaseNormal
	v, err = helper.Get(data, "esHandler")
	if err == nil {
		esHandler = v.(elasticsearchhandler.ElasticsearchHandler)
	}

	// Add some code to avoid reconcile multiple statefullset on same time
	// It avoid to have multiple pod that exit the cluster on same time

	// First, we check if some statefullset need to be upgraded
	// Then we only update Statefullset curretly being upgraded or wait is finished
	for indexExpectedSts, expectedSts := range expectedStatefulsets {
		isFound := false

		// Set ownerReferences on expected object before to diff them
		err = ctrl.SetControllerReference(o, expectedSts, r.Client().Scheme())
		if err != nil {
			return diff, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", expectedSts.GetName())
		}

		for i, currentSts := range copyCurrentStatefulsets {
			// Need compare statefulset
			if currentSts.GetName() == expectedSts.GetName() {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(currentSts, expectedSts, patch.CleanMetadata(), patch.IgnoreStatusFields(), patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing statefulset %s", currentSts.GetName())
				}
				if !patchResult.IsEmpty() {
					updatedSts := patchResult.Patched.(*appv1.StatefulSet)
					diff.AddDiff(fmt.Sprintf("diff %s: %s", updatedSts.Name, string(patchResult.Patch)))
					stsToExpectedUpdated = append(stsToExpectedUpdated, updatedSts)
					logger.Debugf("Need update statefulset %s", updatedSts.Name)
				}

				// Remove items found
				copyCurrentStatefulsets = helper.DeleteItemFromSlice(copyCurrentStatefulsets, i)

				break
			}
		}

		if !isFound {
			sts := expectedStatefulsets[indexExpectedSts]
			// Need create statefulset
			diff.AddDiff(fmt.Sprintf("Statefulset %s not yet exist", sts.GetName()))

			diff.AddObjectToCreate(sts)

			logger.Debugf("Need create statefulset %s", sts.GetName())
		}
	}

	if len(copyCurrentStatefulsets) > 0 {
		for _, sts := range copyCurrentStatefulsets {
			diff.AddDiff(fmt.Sprintf("Need delete statefulset %s", sts.GetName()))
		}
	}

	// Check if on TLS blackout to reconcile all statefulset as the last hope
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
		logger.Info("Detect we are on TLS blackout. Reconcile all statefulset")
		for _, sts := range stsToExpectedUpdated {
			diff.AddObjectToUpdate(sts)
		}
	} else {
		// Check if on current upgrade phase, to only upgrade the statefullset currently being upgraded
		// Or statefullset with current replica to 0 (maybee stop all pod)
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionFalse) {
			// Already on upgrade phase
			logger.Debugf("Detect phase: %s", StatefulsetPhaseUpgrade)

			// Upgrade only one active statefulset or current upgrade
			for _, sts := range currentStatefulsets {

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if localhelper.IsOnStatefulSetUpgradeState(sts) && *sts.Spec.Replicas > 0 && os.Getenv("TEST") != "true" {

					data["phase"] = StatefulsetPhaseUpgrade

					// Check if current statefullset need to be upgraded
					for _, stsNeedUpgraded := range stsToExpectedUpdated {
						if stsNeedUpgraded.GetName() == sts.Name {
							logger.Infof("Detect we need to upgrade Statefullset %s that being already on upgrade state", sts.Name)
							diff.AddObjectToUpdate(stsNeedUpgraded)
							break
						}
					}

					logger.Infof("Phase statefulset upgrade: wait pod %d (upgraded) / %d (ready) on %s", (sts.Status.Replicas - sts.Status.UpdatedReplicas), (sts.Status.Replicas - sts.Status.ReadyReplicas), sts.Name)
				}
			}

			// Allow upgrade no active statefulset
			for _, sts := range currentStatefulsets {

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if *sts.Spec.Replicas == 0 && os.Getenv("TEST") != "true" {
					// Check if current statefullset need to be upgraded
					for _, stsNeedUpgraded := range stsToExpectedUpdated {
						if stsNeedUpgraded.GetName() == sts.Name {
							data["phase"] = StatefulsetPhaseUpgrade
							logger.Infof("Detect we need to upgrade Statefullset %s that not yet active (replica 0)", sts.Name)
							diff.AddObjectToUpdate(stsNeedUpgraded)
							break
						}
					}
				}
			}

			// Update phase if needed
			if data["phase"] != StatefulsetPhaseUpgrade {
				// We need to enable balancing before upgrade
				// We need to retry if error. We can't stay cluster on this state
				// On test we never launch real cluster, so we need to skit this
				if os.Getenv("TEST") != "true" {
					if esHandler == nil {
						return diff, res, errors.New("Elasticsearch handler is nil. We need to get it before continue to have ability to re activate balancing")
					}
					if err = esHandler.EnableRoutingRebalance(); err != nil {
						return diff, res, errors.Wrap(err, "Error when enable routing rebalance")
					}
				}
				data["phase"] = StatefulsetPhaseUpgradeFinished
			}
		} else if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionFalse) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionTrue) {
			// Chain with the next upgrade if needed, to avoid break TLS propagation ...
			// Start upgrade phase
			activeStateFulsetAlreadyUpgraded := false

			for _, sts := range stsToExpectedUpdated {
				if *sts.Spec.Replicas == 0 {
					diff.AddObjectToUpdate(sts)
				} else if !activeStateFulsetAlreadyUpgraded {
					data["phase"] = StatefulsetPhaseUpgradeStarted
					activeStateFulsetAlreadyUpgraded = true
					diff.AddObjectToUpdate(sts)

					// We need to disable balancing before upgrade
					// We not need to block if error
					if esHandler != nil {
						if err = esHandler.DisableRoutingRebalance(); err != nil {
							logger.Warnf("Error when disable routing rebalance: %s", err)
						}
					} else {
						logger.Warn("Elasticsearch not ready. We skip to disable routing rebalance. It something can be normal if you provision the cluster first time.")
					}
				}
			}
		}
	}

	logger.Debugf("Phase after diff: %s", data["phase"])

	diff.SetObjectsToDelete(copyCurrentStatefulsets)

	return diff, res, nil
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *statefulsetReconciler) OnSuccess(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, diff multiphase.MultiPhaseDiff[*appv1.StatefulSet], logger *logrus.Entry) (res reconcile.Result, err error) {
	var d any

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(shared.PhaseName)

	logger.Debugf("Phase on success: %s", phase)

	// Handle TLS blackout
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
		logger.Info("Detect we are on blackout TLS, start to delete all pods")
		podList := &corev1.PodList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client().List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
		}
		if len(podList.Items) > 0 {
			for _, p := range podList.Items {
				if err = r.Client().Delete(ctx, &p); err != nil {
					return res, errors.Wrapf(err, "Error when delete pod %s", p.Name)
				}
				logger.Infof("Successfully delete pod %s", p.Name)
			}
		}
	}

	switch phase {
	case StatefulsetPhaseUpgradeStarted:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition.String(),
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "Statefulsets are being upgraded",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are being upgraded",
		})

		r.Recorder().Eventf(o, corev1.EventTypeNormal, "Completed", "Statefulsets are being upgraded")

		return reconcile.Result{RequeueAfter: time.Second * 30}, nil

	case StatefulsetPhaseUpgrade:
		return reconcile.Result{RequeueAfter: time.Second * 30}, nil

	case StatefulsetPhaseUpgradeFinished:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade.String(),
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "Statefulsets are finished to be upgraded",
		})

		r.Recorder().Eventf(o, corev1.EventTypeNormal, "Completed", "Statefulsets are finished to be upgraded")

		return reconcile.Result{Requeue: true}, nil

	}

	if diff.NeedCreate() || diff.NeedUpdate() || diff.NeedDelete() {
		r.Recorder().Eventf(o, corev1.EventTypeNormal, "Completed", "Statefulsets successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition.String(),
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionFalse) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade.String(),
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "No current upgrade",
		})
	}

	return res, nil
}
