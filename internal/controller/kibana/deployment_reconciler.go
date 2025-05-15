package kibana

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
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
	DeploymentCondition shared.ConditionName = "DeploymentReady"
	DeploymentPhase     shared.PhaseName     = "Deployment"
)

type deploymentReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *appv1.Deployment]
	isOpenshift bool
}

func newDeploymentReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *appv1.Deployment]) {
	return &deploymentReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *appv1.Deployment](
			client,
			DeploymentPhase,
			DeploymentCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing satefulsets
func (r *deploymentReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*appv1.Deployment], res reconcile.Result, err error) {
	dpl := &appv1.Deployment{}
	read = multiphase.NewMultiPhaseRead[*appv1.Deployment]()
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	var es *elasticsearchcrd.Elasticsearch
	configMapsChecksum := make([]*corev1.ConfigMap, 0)
	secretsChecksum := make([]*corev1.Secret, 0)

	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetDeploymentName(o)}, dpl); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read deployment")
		}
		dpl = nil
	}
	if dpl != nil {
		read.AddCurrentObject(dpl)
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

	// Read keystore secret if needed
	if o.Spec.KeystoreSecretRef != nil && o.Spec.KeystoreSecretRef.Name != "" {
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.KeystoreSecretRef.Name}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.KeystoreSecretRef.Name)
			}
			logger.Warnf("Secret %s not yet exist, try again later", o.Spec.KeystoreSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		secretsChecksum = append(secretsChecksum, s)
	}

	// Read APi Crt if needed
	if o.Spec.Tls.IsTlsEnabled() {
		if o.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTls(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTls(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTls(o))
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

	// Read Custom CA Elasticsearch if needed
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
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, kibanacrd.KibanaAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	configMapsChecksum = append(configMapsChecksum, helper.ToSlicePtr(cmList.Items)...)

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

	// Generate expected deployment
	expectedDeployments, err := buildDeployments(o, es, secretsChecksum, configMapsChecksum, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate deployment")
	}
	read.SetExpectedObjects(expectedDeployments)

	return read, res, nil
}
