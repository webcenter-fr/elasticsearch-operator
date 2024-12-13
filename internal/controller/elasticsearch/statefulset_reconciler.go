package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
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
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newStatefulsetReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &statefulsetReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			StatefulsetPhase,
			StatefulsetCondition,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
	}
}

// Read existing satefulsets
func (r *statefulsetReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	stsList := &appv1.StatefulSetList{}
	read = controller.NewBasicMultiPhaseRead()
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	configMapsChecksum := make([]corev1.ConfigMap, 0)
	secretsChecksum := make([]corev1.Secret, 0)

	// Read current satefulsets
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read statefulset")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(stsList.Items))

	// Read keystore secret if needed
	if o.Spec.GlobalNodeGroup.KeystoreSecretRef != nil && o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name != "" {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, *s)
	}

	// Read cacerts secret if needed
	if o.Spec.GlobalNodeGroup.CacertsSecretRef != nil && o.Spec.GlobalNodeGroup.CacertsSecretRef.Name != "" {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.GlobalNodeGroup.CacertsSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.GlobalNodeGroup.CacertsSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.GlobalNodeGroup.CacertsSecretRef.Name)
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, *s)
	}

	// Read API certificate secret if needed
	if o.Spec.Tls.IsTlsEnabled() {
		if o.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsApi(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsApi(o))
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsApi(o))
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		} else {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		}
	}

	// Read transport certicate secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsTransport(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsTransport(o))
		}
		r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsTransport(o))
		return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	secretsChecksum = append(secretsChecksum, *s)

	// Read configMaps to generate checksum
	// Keep only configmap of type config
	labelSelectors, err = labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	for _, cm := range cmList.Items {
		if cm.Annotations[fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey)] == "config" {
			configMapsChecksum = append(configMapsChecksum, cm)
		}
	}

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.GlobalNodeGroup.AdditionalVolumes {
		if v.ConfigMap != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.ConfigMap.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", v.ConfigMap.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", v.ConfigMap.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}

		if v.Secret != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Secret.SecretName}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", v.Secret.SecretName)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", v.Secret.SecretName)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
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
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.SecretKeyRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", env.ValueFrom.SecretKeyRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", env.ValueFrom.SecretKeyRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", env.ValueFrom.ConfigMapKeyRef.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", env.ValueFrom.ConfigMapKeyRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Read extra Env from to generate checksum if secret or configmap
	for _, ef := range envFromList {
		if ef.SecretRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.SecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", ef.SecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", ef.SecretRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if ef.ConfigMapRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.ConfigMapRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", ef.ConfigMapRef.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", ef.ConfigMapRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Generate expected statefulsets
	expectedSts, err := buildStatefulsets(o, secretsChecksum, configMapsChecksum)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate statefulsets")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedSts))

	return read, res, nil
}

// Diff permit to check if statefulset is up to date
func (r *statefulsetReconciler) Diff(ctx context.Context, resource object.MultiPhaseObject, read controller.MultiPhaseRead, data map[string]any, ignoreDiff ...patch.CalculateOption) (diff controller.MultiPhaseDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	currentStatefulsets := read.GetCurrentObjects()
	copyCurrentStatefulsets := make([]client.Object, len(currentStatefulsets))
	copy(copyCurrentStatefulsets, currentStatefulsets)

	expectedStatefulsets := read.GetExpectedObjects()

	diff = controller.NewBasicMultiPhaseDiff()

	stsToUpdate := make([]client.Object, 0)
	stsToExpectedUpdated := make([]*appv1.StatefulSet, 0)
	stsToCreate := make([]client.Object, 0)
	data["phase"] = StatefulsetPhaseNormal

	// Add some code to avoid reconcile multiple statefullset on same time
	// It avoid to have multiple pod that exit the cluster on same time

	// First, we check if some statefullset need to be upgraded
	// Then we only update Statefullset curretly being upgraded or wait is finished
	for indexExpectedSts, expectedSts := range expectedStatefulsets {
		isFound := false
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
					r.Log.Debugf("Need update statefulset %s", updatedSts.Name)
				}

				// Remove items found
				copyCurrentStatefulsets = helper.DeleteItemFromSlice(copyCurrentStatefulsets, i).([]client.Object)

				break
			}
		}

		if !isFound {
			sts := expectedStatefulsets[indexExpectedSts]
			// Need create statefulset
			diff.AddDiff(fmt.Sprintf("Statefulset %s not yet exist", sts.GetName()))

			stsToCreate = append(stsToCreate, sts)

			r.Log.Debugf("Need create statefulset %s", sts.GetName())
		}
	}

	if len(copyCurrentStatefulsets) > 0 {
		for _, sts := range copyCurrentStatefulsets {
			diff.AddDiff(fmt.Sprintf("Need delete statefulset %s", sts.GetName()))
		}
	}

	// Check if on TLS blackout to reconcile all statefulset as the last hope
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
		r.Log.Info("Detect we are on TLS blackout. Reconcile all statefulset")
		for _, sts := range stsToExpectedUpdated {
			stsToUpdate = append(stsToUpdate, sts)
		}
	} else {
		// Check if on current upgrade phase, to only upgrade the statefullset currently being upgraded
		// Or statefullset with current replica to 0 (maybee stop all pod)
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionFalse) {
			// Already on upgrade phase
			r.Log.Debugf("Detect phase: %s", StatefulsetPhaseUpgrade)

			// Upgrade only one active statefulset or current upgrade
			for _, object := range currentStatefulsets {
				sts := object.(*appv1.StatefulSet)

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if localhelper.IsOnStatefulSetUpgradeState(sts) && *sts.Spec.Replicas > 0 && os.Getenv("TEST") != "true" {

					data["phase"] = StatefulsetPhaseUpgrade

					// Check if current statefullset need to be upgraded
					for _, stsNeedUpgraded := range stsToExpectedUpdated {
						if stsNeedUpgraded.GetName() == sts.Name {
							r.Log.Infof("Detect we need to upgrade Statefullset %s that being already on upgrade state", sts.Name)
							stsToUpdate = append(stsToUpdate, stsNeedUpgraded)
							break
						}
					}

					r.Log.Infof("Phase statefulset upgrade: wait pod %d (upgraded) / %d (ready) on %s", (sts.Status.Replicas - sts.Status.UpdatedReplicas), (sts.Status.Replicas - sts.Status.ReadyReplicas), sts.Name)
				}
			}

			// Allow upgrade no active statefulset
			for _, object := range currentStatefulsets {
				sts := object.(*appv1.StatefulSet)

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if *sts.Spec.Replicas == 0 && os.Getenv("TEST") != "true" {
					// Check if current statefullset need to be upgraded
					for _, stsNeedUpgraded := range stsToExpectedUpdated {
						if stsNeedUpgraded.GetName() == sts.Name {
							data["phase"] = StatefulsetPhaseUpgrade
							r.Log.Infof("Detect we need to upgrade Statefullset %s that not yet active (replica 0)", sts.Name)
							stsToUpdate = append(stsToUpdate, stsNeedUpgraded)
							break
						}
					}
				}
			}

			// Update phase if needed
			if data["phase"] != StatefulsetPhaseUpgrade {
				data["phase"] = StatefulsetPhaseUpgradeFinished
			}
		} else if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionFalse) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionTrue) {
			// Chain with the next upgrade if needed, to avoid break TLS propagation ...
			// Start upgrade phase
			activeStateFulsetAlreadyUpgraded := false

			for _, sts := range stsToExpectedUpdated {
				if *sts.Spec.Replicas == 0 {
					stsToUpdate = append(stsToUpdate, sts)
				} else if !activeStateFulsetAlreadyUpgraded {
					data["phase"] = StatefulsetPhaseUpgradeStarted
					activeStateFulsetAlreadyUpgraded = true
					stsToUpdate = append(stsToUpdate, sts)
				}
			}
		}
	}

	r.Log.Debugf("Phase after diff: %s", data["phase"])

	diff.SetObjectsToCreate(stsToCreate)
	diff.SetObjectsToUpdate(stsToUpdate)
	diff.SetObjectsToDelete(copyCurrentStatefulsets)

	return diff, res, nil
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *statefulsetReconciler) OnSuccess(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, diff controller.MultiPhaseDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(shared.PhaseName)

	r.Log.Debugf("Phase on success: %s", phase)

	// Handle TLS blackout
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
		r.Log.Info("Detect we are on blackout TLS, start to delete all pods")
		podList := &corev1.PodList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
		if err != nil {
			return res, errors.Wrap(err, "Error when generate label selector")
		}
		if err = r.Client.List(ctx, podList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}, &client.ListOptions{}); err != nil {
			return res, errors.Wrapf(err, "Error when read Elasticsearch pods")
		}
		if len(podList.Items) > 0 {
			for _, p := range podList.Items {
				if err = r.Client.Delete(ctx, &p); err != nil {
					return res, errors.Wrapf(err, "Error when delete pod %s", p.Name)
				}
				r.Log.Infof("Successfully delete pod %s", p.Name)
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

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are being upgraded")

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil

	case StatefulsetPhaseUpgrade:
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil

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

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are finished to be upgraded")

		return ctrl.Result{Requeue: true}, nil

	}

	if diff.NeedCreate() || diff.NeedUpdate() || diff.NeedDelete() {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets successfully updated")
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
