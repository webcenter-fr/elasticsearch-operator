package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type statefulsetPhase string

const (
	StatefulsetCondition        = "StatefulsetReady"
	StatefulsetConditionUpgrade = "StatefulsetUpgrade"
	StatefulsetPhase            = "Statefullset"
)

var (
	phaseStsUpgradeStarted  statefulsetPhase = "statefulsetUpgradeStarted"
	phaseStsUpgrade         statefulsetPhase = "statefulsetUpgrade"
	phaseStsUpgradeFinished statefulsetPhase = "statefulsetUpgradeFinished"
	phaseStsNormal          statefulsetPhase = "statefulsetNormal"
)

type StatefulsetReconciler struct {
	common.Reconciler
}

func NewStatefulsetReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &StatefulsetReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "statefulset",
			}),
			Name:   "statefulset",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *StatefulsetReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, StatefulsetCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   StatefulsetCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	o.Status.Phase = StatefulsetPhase

	return res, nil
}

// Read existing satefulsets
func (r *StatefulsetReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	stsList := &appv1.StatefulSetList{}
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	configMapsChecksum := make([]corev1.ConfigMap, 0)
	secretsChecksum := make([]corev1.Secret, 0)

	// Read current satefulsets
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, stsList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read statefulset")
	}
	data["currentStatefulsets"] = stsList.Items

	// Read keystore secret if needed
	if o.Spec.GlobalNodeGroup.KeystoreSecretRef != nil && o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name != "" {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			}
			r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, *s)
	}

	// Read API certificate secret if needed
	if o.IsTlsApiEnabled() {
		if o.IsSelfManagedSecretForTlsApi() {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsApi(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsApi(o))
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsApi(o))
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		} else {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		}
	}

	// Read transport certicate secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsTransport(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsTransport(o))
		}
		r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsTransport(o))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	secretsChecksum = append(secretsChecksum, *s)

	// Read configMaps to generate checksum
	labelSelectors, err = labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read configMap")
	}
	configMapsChecksum = append(configMapsChecksum, cmList.Items...)

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.GlobalNodeGroup.AdditionalVolumes {
		if v.ConfigMap != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", v.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", v.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}

		if v.Secret != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", v.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", v.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
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
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.SecretKeyRef.LocalObjectReference.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", env.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", env.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Read extra Env from to generate checksum if secret or configmap
	for _, ef := range envFromList {
		if ef.SecretRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.SecretRef.LocalObjectReference.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", ef.SecretRef.LocalObjectReference.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", ef.SecretRef.LocalObjectReference.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if ef.ConfigMapRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.ConfigMapRef.LocalObjectReference.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", ef.ConfigMapRef.LocalObjectReference.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", ef.ConfigMapRef.LocalObjectReference.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Generate expected statefulsets
	expectedSts, err := BuildStatefulsets(o, secretsChecksum, configMapsChecksum)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate statefulsets")
	}
	data["expectedStatefulsets"] = expectedSts

	return res, nil
}

// Diff permit to check if statefulset is up to date
func (r *StatefulsetReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentStatefulsets")
	if err != nil {
		return diff, res, err
	}
	currentStatefulsets := d.([]appv1.StatefulSet)

	copyCurrentStatefulsets := make([]appv1.StatefulSet, len(currentStatefulsets))
	copy(copyCurrentStatefulsets, currentStatefulsets)

	d, err = helper.Get(data, "expectedStatefulsets")
	if err != nil {
		return diff, res, err
	}
	expectedStatefulsets := d.([]appv1.StatefulSet)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	stsToUpdate := make([]client.Object, 0)
	stsToExpectedUpdated := make([]*appv1.StatefulSet, 0)
	stsToCreate := make([]client.Object, 0)
	data["phase"] = phaseStsNormal

	// Add some code to avoid reconcile multiple statefullset on same time
	// It avoid to have multiple pod that exit the cluster on same time

	// First, we check if some statefullset need to be upgraded
	// Then we only update Statefullset curretly being upgraded or wait is finished
	for indexExpectedSts, expectedSts := range expectedStatefulsets {
		isFound := false
		for i, currentSts := range copyCurrentStatefulsets {
			// Need compare statefulset
			if currentSts.Name == expectedSts.Name {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentSts, &expectedSts, patch.CleanMetadata(), patch.IgnoreStatusFields(), patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing statefulset %s", currentSts.Name)
				}
				if !patchResult.IsEmpty() {
					updatedSts := patchResult.Patched.(*appv1.StatefulSet)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedSts.Name, string(patchResult.Patch)))
					stsToExpectedUpdated = append(stsToExpectedUpdated, updatedSts)
					r.Log.Debugf("Need update statefulset %s", updatedSts.Name)
				}

				// Remove items found
				copyCurrentStatefulsets = helperdiff.DeleteItemFromSlice(copyCurrentStatefulsets, i).([]appv1.StatefulSet)

				break
			}
		}

		if !isFound {
			sts := &expectedStatefulsets[indexExpectedSts]
			// Need create statefulset
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Statefulset %s not yet exist\n", sts.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, sts, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(sts); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on statefulset %s", sts.Name)
			}

			stsToCreate = append(stsToCreate, sts)

			r.Log.Debugf("Need create statefulset %s", sts.Name)
		}
	}

	if len(copyCurrentStatefulsets) > 0 {
		diff.NeedDelete = true
		for _, sts := range copyCurrentStatefulsets {
			diff.Diff.WriteString(fmt.Sprintf("Need delete statefulset %s\n", sts.Name))
		}
	}

	// Check if on TLS blackout to reconcile all statefulset as the last hope
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout, metav1.ConditionTrue) {
		r.Log.Info("Detect we are on TLS blackout. Reconcile all statefulset")
		for _, sts := range stsToExpectedUpdated {
			stsToUpdate = append(stsToUpdate, sts)
		}
	} else {
		// Check if on current upgrade phase, to only upgrade the statefullset currently being upgraded
		// Or statefullset with current replica to 0 (maybee stop all pod)
		if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition, metav1.ConditionFalse) {
			// Already on upgrade phase
			r.Log.Debugf("Detect phase: %s", phaseStsUpgrade)

			// Upgrade only one active statefulset or current upgrade
			for _, sts := range currentStatefulsets {

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if localhelper.IsOnStatefulSetUpgradeState(&sts) && *sts.Spec.Replicas > 0 && os.Getenv("TEST") != "true" {

					data["phase"] = phaseStsUpgrade

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
			for _, sts := range currentStatefulsets {

				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if *sts.Spec.Replicas == 0 && os.Getenv("TEST") != "true" {
					// Check if current statefullset need to be upgraded
					for _, stsNeedUpgraded := range stsToExpectedUpdated {
						if stsNeedUpgraded.GetName() == sts.Name {
							data["phase"] = phaseStsUpgrade
							r.Log.Infof("Detect we need to upgrade Statefullset %s that not yet active (replica 0)", sts.Name)
							stsToUpdate = append(stsToUpdate, stsNeedUpgraded)
							break
						}
					}
				}
			}

			// Update phase if needed
			if data["phase"] != phaseStsUpgrade {
				data["phase"] = phaseStsUpgradeFinished
			}
		} else if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionFalse) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition, metav1.ConditionTrue) {
			// Start upgrade phase
			activeStateFulsetAlreadyUpgraded := false

			for _, sts := range stsToExpectedUpdated {
				if *sts.Spec.Replicas == 0 {
					stsToUpdate = append(stsToUpdate, sts)
				} else if !activeStateFulsetAlreadyUpgraded {
					data["phase"] = phaseStsUpgradeStarted
					activeStateFulsetAlreadyUpgraded = true
					stsToUpdate = append(stsToUpdate, sts)
				}
			}
		}
	}

	r.Log.Debugf("Phase after diff: %s", data["phase"])

	data["listToCreate"] = stsToCreate
	data["listToUpdate"] = stsToUpdate
	data["listToDelete"] = helperdiff.ToSliceOfObject(copyCurrentStatefulsets)

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *StatefulsetReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    StatefulsetCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *StatefulsetReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var (
		d any
	)

	d, err = helper.Get(data, "phase")
	if err != nil {
		return res, err
	}
	phase := d.(statefulsetPhase)

	r.Log.Debugf("Phase on success: %s", phase)

	// Handle TLS blackout
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout, metav1.ConditionTrue) {
		r.Log.Info("Detect we are on blackout TLS, start to delete all pods")
		podList := &corev1.PodList{}
		labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
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
	case phaseStsUpgradeStarted:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "Statefulsets are being upgraded",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are being upgraded",
		})

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are being upgraded")

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil

	case phaseStsUpgrade:
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil

	case phaseStsUpgradeFinished:
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Statefulsets are ready",
		})

		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "Statefulsets are finished to be upgraded",
		})

		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets are finished to be upgraded")

		return ctrl.Result{Requeue: true}, nil

	}

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulsets successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade, metav1.ConditionFalse) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    StatefulsetConditionUpgrade,
			Reason:  "Success",
			Status:  metav1.ConditionFalse,
			Message: "No current upgrade",
		})
	}

	return res, nil
}
