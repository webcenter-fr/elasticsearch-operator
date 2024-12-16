package logstash

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatefulsetCondition shared.ConditionName = "StatefulsetReady"
	StatefulsetPhase     shared.PhaseName     = "Statefulset"
)

type statefulsetReconciler struct {
	controller.MultiPhaseStepReconcilerAction
	isOpenshift bool
}

func newStatefulsetReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &statefulsetReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			StatefulsetPhase,
			StatefulsetCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing satefulsets
func (r *statefulsetReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)
	sts := &appv1.StatefulSet{}
	read = controller.NewBasicMultiPhaseRead()
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	var es *elasticsearchcrd.Elasticsearch
	configMapsChecksum := make([]corev1.ConfigMap, 0)
	secretsChecksum := make([]corev1.Secret, 0)

	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read statefulset")
		}
		sts = nil
	}
	if sts != nil {
		read.SetCurrentObjects([]client.Object{sts})
	}

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {
		es, err = common.GetElasticsearchFromRef(ctx, r.Client(), o, o.Spec.ElasticsearchRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read ElasticsearchRef")
		}
		if es == nil {
			logger.Warn("ElasticsearchRef not found, try latter")
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		es = nil
	}

	// Read keystore secret to generate checksum
	if o.Spec.KeystoreSecretRef != nil && o.Spec.KeystoreSecretRef.Name != "" {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.KeystoreSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.KeystoreSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.KeystoreSecretRef.Name)
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, *s)
	}

	// Read Custom CA Elasticsearch to generate checksum
	if (o.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled()) || o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		if o.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForCAElasticsearch(o))
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		} else {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			if len(s.Data["ca.crt"]) == 0 {
				return read, res, errors.Errorf("Secret %s must have a key `ca.crt`", s.Name)
			}

			secretsChecksum = append(secretsChecksum, *s)
		}
	}

	// Read configMaps to generate checksum
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, logstashcrd.LogstashAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMaps")
	}
	configMapsChecksum = append(configMapsChecksum, cmList.Items...)

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.Deployment.AdditionalVolumes {
		if v.ConfigMap != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.ConfigMap.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", v.ConfigMap.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", v.ConfigMap.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}

		if v.Secret != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Secret.SecretName}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", v.Secret.SecretName)
				}
				logger.Warnf("Secret %s not yet exist, try again later", v.Secret.SecretName)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
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
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", env.ValueFrom.ConfigMapKeyRef.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", env.ValueFrom.ConfigMapKeyRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
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
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if ef.ConfigMapRef != nil {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.ConfigMapRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read configMap %s", ef.ConfigMapRef.Name)
				}
				logger.Warnf("ConfigMap %s not yet exist, try again later", ef.ConfigMapRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
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
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, *s)
	}

	// Generate expected statefulset
	expectedSts, err := buildStatefulsets(o, es, secretsChecksum, configMapsChecksum, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate statefulset")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedSts))

	return read, res, nil
}

func (r *statefulsetReconciler) GetIgnoresDiff() []patch.CalculateOption {
	return []patch.CalculateOption{
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
	}
}
