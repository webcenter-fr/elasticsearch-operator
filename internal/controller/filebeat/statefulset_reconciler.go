package filebeat

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	StatefulsetCondition shared.ConditionName = "StatefulsetReady"
	StatefulsetPhase     shared.PhaseName     = "Statefulset"
)

type statefulsetReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *appv1.StatefulSet]
	isOpenshift bool
}

func newStatefulsetReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *appv1.StatefulSet]) {
	return &statefulsetReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *appv1.StatefulSet](
			client,
			StatefulsetPhase,
			StatefulsetCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing satefulsets
func (r *statefulsetReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*appv1.StatefulSet], res reconcile.Result, err error) {
	sts := &appv1.StatefulSet{}
	read = multiphase.NewMultiPhaseRead[*appv1.StatefulSet]()
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	configMapsChecksum := make([]*corev1.ConfigMap, 0)
	secretsChecksum := make([]*corev1.Secret, 0)
	var cms []*corev1.ConfigMap

	var (
		es *elasticsearchcrd.Elasticsearch
		ls *logstashcrd.Logstash
	)

	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read statefulset")
		}
		sts = nil
	}
	if sts != nil {
		read.AddCurrentObject(sts)
	}

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {
		es, err = common.GetElasticsearchFromRef(ctx, r.Client(), o, o.Spec.ElasticsearchRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read ElasticsearchRef")
		}
		if es == nil {
			logger.Warn("ElasticsearchRef not found, try latter")
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		es = nil
	}

	// Read Logstash
	if o.Spec.LogstashRef.IsManaged() {
		ls = &logstashcrd.Logstash{}
		namespace := o.Namespace
		if o.Spec.LogstashRef.ManagedLogstashRef.Namespace != "" {
			namespace = o.Spec.LogstashRef.ManagedLogstashRef.Namespace
		}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: namespace, Name: o.Spec.LogstashRef.ManagedLogstashRef.Name}, ls); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read logstash %s", o.Spec.LogstashRef.ManagedLogstashRef.Name)
			}
			logger.Warnf("Logstash %s not yet exist, try again later", o.Spec.LogstashRef.ManagedLogstashRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Read reflected secret to add on checksum
		if ls.Spec.Pki.IsEnabled() && ls.Spec.Pki.HasBeatCertificate() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCALogstash(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCALogstash(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForCALogstash(o))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
		}

	}

	// Read Custom CA Elasticsearch to generate checksum
	if (o.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled()) || o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		if o.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForCAElasticsearch(o))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, s)
		} else {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			if len(s.Data["ca.crt"]) == 0 {
				return read, res, errors.Errorf("Secret %s must have a key `ca.crt`", s.Name)
			}

			secretsChecksum = append(secretsChecksum, s)
		}
	}

	// Read custom CA for Logstash to add on checksum
	if o.Spec.LogstashRef.LogstashCaSecretRef != nil {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LogstashRef.LogstashCaSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Read Elasticsearch secretRef to add on checksum
	if o.Spec.ElasticsearchRef.SecretRef != nil {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.SecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.SecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.ElasticsearchRef.SecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Read configMaps to generate checksum
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, beatcrd.FilebeatAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	cms = helper.ToSlicePtr(cmList.Items)
	configMapsChecksum = append(configMapsChecksum, cms...)

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.Deployment.AdditionalVolumes {
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

	// Read extra Env to generate checksum if secret or configmap
	for _, env := range o.Spec.Deployment.Env {
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
	for _, ef := range o.Spec.Deployment.EnvFrom {
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

	// Read certificates
	if o.Spec.Pki.IsEnabled() {

		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTls(o)}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTls(o))
			}
			logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTls(o))
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Generate expected statefulset
	expectedSts, err := buildStatefulsets(o, es, ls, cms, secretsChecksum, configMapsChecksum, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate statefulset")
	}
	read.SetExpectedObjects(expectedSts)

	return read, res, nil
}

func (r *statefulsetReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
	}
}
