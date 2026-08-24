package elasticsearch

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"emperror.dev/errors"
	elasticsearchhandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/certificate"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	ssa "github.com/disaster37/operator-sdk-extra/v3/pkg/helper/ssa"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
	multiphase.MultiPhaseStepReconcilerActionWithDiff[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet]
	isOpenshift bool
}

func newStatefulsetReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerActionWithDiff[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet]) {
	return &statefulsetReconciler{
		MultiPhaseStepReconcilerActionWithDiff: multiphase.NewMultiPhaseStepReconcilerActionWithDiff[*elasticsearchcrd.Elasticsearch, *appv1.StatefulSet](
			client,
			StatefulsetPhase,
			StatefulsetCondition,
			recorder,
			common.FieldManager,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing satefulsets
func (r *statefulsetReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*appv1.StatefulSet], res reconcile.Result, err error) {
	stsList := &appv1.StatefulSetList{}
	read = multiphase.NewMultiPhaseRead[*appv1.StatefulSet]()
	var s *corev1.Secret
	var cm *corev1.ConfigMap
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
	// The SSA dry-run diff (multiphase.ClassifyObjects) does not normalize the
	// pod-template creationTimestamp the API server sets on live StatefulSets,
	// so strip it here to avoid a perpetual "update" classification.
	for i := range stsList.Items {
		stsList.Items[i].Spec.Template.ObjectMeta.CreationTimestamp = metav1.Time{}
	}
	read.SetCurrentObjects(helper.ToSlicePtr(stsList.Items))

	// Read keystore secret if needed
	if o.Spec.GlobalNodeGroup.KeystoreSecretRef != nil && o.Spec.GlobalNodeGroup.KeystoreSecretRef.Name != "" {
		s = &corev1.Secret{}
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
		s = &corev1.Secret{}
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
			s = &corev1.Secret{}
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsApi(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsApi(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsApi(o))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
		} else {
			s = &corev1.Secret{}
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
	s = &corev1.Secret{}
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTlsTransport(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTlsTransport(o))
		}
		logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTlsTransport(o))
		return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}
	transportSecret := s.DeepCopy()
	secretsChecksum = append(secretsChecksum, s)

	// Compute the transport rollout marker from the TLS saga's LayerSignals.
	// The marker lives ONLY on the StatefulSet pod-template, and changes only
	// when the saga signals a real CA/leaf rotation (never on node add/remove).
	transportMarker := ""
	sig, _ := data["tls.TlsTransport"].(*certificate.LayerSignals)
	shouldRollout := certificate.ShouldRollout(certificate.RolloutOnAdditive, sig)

	markerKey := fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, GetSecretNameForTlsTransport(o))
	currentMarker := ""
	for i := range stsList.Items {
		if v, ok := stsList.Items[i].Spec.Template.Annotations[markerKey]; ok && v != "" {
			currentMarker = v
			break
		}
	}

	if shouldRollout || currentMarker == "" {
		// Fresh marker: at rotation/bootstrap every node cert + CA changed, so a
		// whole-Secret hash is a correct, stable marker.
		h, hErr := certificate.SecretHash(transportSecret)
		if hErr != nil {
			return read, res, errors.Wrap(hErr, "Error when hash transport secret")
		}
		transportMarker = h
	} else {
		// No rollout: keep the current marker so the pod-template is unchanged.
		transportMarker = currentMarker
	}

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
			cm = &corev1.ConfigMap{}
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
			s = &corev1.Secret{}
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
			s = &corev1.Secret{}
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
			cm = &corev1.ConfigMap{}
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
			s = &corev1.Secret{}
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
			cm = &corev1.ConfigMap{}
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
	expectedSts, err := buildStatefulsets(o, secretsChecksum, configMapsChecksum, r.isOpenshift, transportMarker)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate statefulsets")
	}
	for _, sts := range expectedSts {
		if err = ctrl.SetControllerReference(o, sts, r.Client().Scheme()); err != nil {
			return read, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", sts.GetName())
		}
	}
	common.InjectTypeMeta(r.Client().Scheme(), expectedSts...)
	read.SetExpectedObjects(expectedSts)

	return read, res, nil
}

// Diff classify the expected statefulsets using SSA dry-run and apply the
// "only one StatefulSet upgraded at a time" gating logic.
func (r *statefulsetReconciler) Diff(ctx context.Context, o *elasticsearchcrd.Elasticsearch, read multiphase.MultiPhaseRead[*appv1.StatefulSet], data map[string]any, logger *logrus.Entry) (diff multiphase.MultiPhaseDiff[*appv1.StatefulSet], res reconcile.Result, err error) {
	var esHandler elasticsearchhandler.ElasticsearchHandler
	var v any

	currentStatefulsets := read.GetCurrentObjects()
	expectedStatefulsets := read.GetExpectedObjects()

	data["phase"] = StatefulsetPhaseNormal
	v, err = helper.Get(data, "esHandler")
	if err == nil {
		esHandler = v.(elasticsearchhandler.ElasticsearchHandler)
	}

	// Classify expected objects using SSA dry-run. Unchanged objects are
	// silently skipped (no perpetual apply loop). The library's
	// multiphase.ClassifyObjects does not normalize the pod-template
	// creationTimestamp the API server injects into SSA dry-run responses, so
	// classify manually here with that field stripped.
	creates, updates, deletes, diffStrs, err := classifyStatefulsets(ctx, r.Client(), expectedStatefulsets, currentStatefulsets, common.FieldManager)
	if err != nil {
		return diff, res, err
	}

	// Add some code to avoid reconcile multiple statefullset on same time
	// It avoid to have multiple pod that exit the cluster on same time
	if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, TlsConditionBlackout.String(), metav1.ConditionTrue) {
		// On TLS blackout, reconcile all statefulset as the last hope
		logger.Info("Detect we are on TLS blackout. Reconcile all statefulset")
	} else if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionTrue) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionFalse) {
		// Already on upgrade phase: only upgrade the statefulset currently
		// being upgraded or statefulset with 0 replica.
		logger.Debugf("Detect phase: %s", StatefulsetPhaseUpgrade)

		if common.IsEnvtest() {
			// No kubelet in envtest: apply all pending StatefulSets immediately
			// and mark the upgrade finished, otherwise the gating would loop
			// forever waiting for pods that never roll.
			data["phase"] = StatefulsetPhaseUpgradeFinished
		} else {
			updatesAllowed := make([]*appv1.StatefulSet, 0, len(updates))
			for _, sts := range currentStatefulsets {
				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if localhelper.IsOnStatefulSetUpgradeState(sts) && *sts.Spec.Replicas > 0 {
					data["phase"] = StatefulsetPhaseUpgrade

					for _, stsNeedUpgraded := range updates {
						if stsNeedUpgraded.GetName() == sts.Name {
							logger.Infof("Detect we need to upgrade Statefullset %s that being already on upgrade state", sts.Name)
							updatesAllowed = append(updatesAllowed, stsNeedUpgraded)
							break
						}
					}

					logger.Infof("Phase statefulset upgrade: wait pod %d (upgraded) / %d (ready) on %s", (sts.Status.Replicas - sts.Status.UpdatedReplicas), (sts.Status.Replicas - sts.Status.ReadyReplicas), sts.Name)
				}
			}

			for _, sts := range currentStatefulsets {
				// Not found a way to detect that we are on envtest, so without kubelet. We use env TEST to to that.
				// It avoid to stuck test on this phase
				if *sts.Spec.Replicas == 0 {
					for _, stsNeedUpgraded := range updates {
						if stsNeedUpgraded.GetName() == sts.Name {
							data["phase"] = StatefulsetPhaseUpgrade
							logger.Infof("Detect we need to upgrade Statefullset %s that not yet active (replica 0)", sts.Name)
							updatesAllowed = append(updatesAllowed, stsNeedUpgraded)
							break
						}
					}
				}
			}

			if data["phase"] != StatefulsetPhaseUpgrade {
				// We need to enable balancing before upgrade
				// We need to retry if error. We can't stay cluster on this state
				if esHandler == nil {
					return diff, res, errors.New("Elasticsearch handler is nil. We need to get it before continue to have ability to re activate balancing")
				}
				if err = esHandler.EnableRoutingRebalance(); err != nil {
					return diff, res, errors.Wrap(err, "Error when enable routing rebalance")
				}
				data["phase"] = StatefulsetPhaseUpgradeFinished
			}
			updates = updatesAllowed
		}
	} else if condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetConditionUpgrade.String(), metav1.ConditionFalse) && condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, StatefulsetCondition.String(), metav1.ConditionTrue) {
		// Chain with the next upgrade if needed, to avoid break TLS propagation ...
		// Start upgrade phase
		activeStateFulsetAlreadyUpgraded := false
		updatesAllowed := make([]*appv1.StatefulSet, 0, len(updates))

		for _, sts := range updates {
			if *sts.Spec.Replicas == 0 {
				updatesAllowed = append(updatesAllowed, sts)
			} else if !activeStateFulsetAlreadyUpgraded {
				data["phase"] = StatefulsetPhaseUpgradeStarted
				activeStateFulsetAlreadyUpgraded = true
				updatesAllowed = append(updatesAllowed, sts)

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
		updates = updatesAllowed
	}

	logger.Debugf("Phase after diff: %s", data["phase"])

	diff = multiphase.NewMultiPhaseDiff[*appv1.StatefulSet]()
	multiphase.PopulateDiff(diff, creates, updates, deletes, diffStrs)

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

		// In envtest there is no kubelet to roll pods; requeue immediately so
		// the upgrade gating converges within the test timeout.
		if common.IsEnvtest() {
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{RequeueAfter: time.Second * 30}, nil

	case StatefulsetPhaseUpgrade:
		if common.IsEnvtest() {
			return reconcile.Result{Requeue: true}, nil
		}
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

// classifyStatefulsets classifies expected StatefulSets into create/update/
// delete lists using an SSA dry-run, mirroring multiphase.ClassifyObjects but
// additionally stripping the pod-template metadata.creationTimestamp that the
// API server injects into dry-run responses (it is absent on live objects).
func classifyStatefulsets(ctx context.Context, c client.Client, expectedObjects, currentObjects []*appv1.StatefulSet, fieldManager string) (creates, updates, deletes []*appv1.StatefulSet, diffStrs []string, err error) {
	currentByName := make(map[string]*appv1.StatefulSet, len(currentObjects))
	for _, current := range currentObjects {
		currentByName[current.GetName()] = current
	}

	for _, expected := range expectedObjects {
		current, exists := currentByName[expected.GetName()]
		if !exists {
			creates = append(creates, expected)
			diffStrs = append(diffStrs, fmt.Sprintf("Create object '%s'", expected.GetName()))
			continue
		}
		delete(currentByName, expected.GetName())

		changed, dErr := statefulsetDiff(ctx, c, expected, current, fieldManager)
		if dErr != nil {
			return nil, nil, nil, nil, dErr
		}
		if changed {
			updates = append(updates, expected)
			diffStrs = append(diffStrs, fmt.Sprintf("Update object '%s'", expected.GetName()))
		}
	}

	for _, current := range currentByName {
		deletes = append(deletes, current)
		diffStrs = append(diffStrs, fmt.Sprintf("Need delete object '%s'", current.GetName()))
	}

	return creates, updates, deletes, diffStrs, nil
}

// statefulsetDiff returns true when the expected StatefulSet differs from the
// current one according to an SSA dry-run, ignoring API-server generated
// fields (including the nested pod-template creationTimestamp).
func statefulsetDiff(ctx context.Context, c client.Client, expected, current *appv1.StatefulSet, fieldManager string) (bool, error) {
	predicted, err := ssa.DryRunApply(ctx, c, expected, fieldManager)
	if err != nil {
		return false, err
	}

	cu, err := runtime.DefaultUnstructuredConverter.ToUnstructured(current)
	if err != nil {
		return false, err
	}
	ssa.Normalize(&unstructured.Unstructured{Object: cu})
	ssa.Normalize(predicted)
	unstructured.RemoveNestedField(cu, "spec", "template", "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(predicted.Object, "spec", "template", "metadata", "creationTimestamp")

	return !reflect.DeepEqual(cu, predicted.Object), nil
}
