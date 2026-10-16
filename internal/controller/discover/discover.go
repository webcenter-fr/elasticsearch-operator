package discover

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/object"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// IsDiscoverSecret permit to know if the secret is managed by discover
func IsDiscoverSecret(s *corev1.Secret) bool {
	if s.Annotations != nil && s.Annotations[discovercrd.DiscoverAnnotationKey] == "true" {
		return true
	}

	return false
}

// IsDiscoverSecretEnv permit to know if the secret is a discover env secret
func IsDiscoverSecretEnv(s *corev1.Secret) bool {
	if IsDiscoverSecret(s) && s.Annotations[fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey)] == "env" {
		return true
	}

	return false
}

// IsDiscoverSecretFile permit to know if the secret is a discover file secret
func IsDiscoverSecretFile(s *corev1.Secret) bool {
	if IsDiscoverSecret(s) && s.Annotations[fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey)] == "file" {
		return true
	}

	return false
}

// GetDiscoverMountPathFromAnnotations permit to get discover mount path from secret annotations
func GetDiscoverMountPathFromAnnotations(s *corev1.Secret) string {
	return s.Annotations[fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey)]
}

// LoadCertFromPem loads a x509 certificate from PEM encoded data
func LoadCertFromPem(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

// ReadDiscoversSecrets permit to read all secrets related to discovers
func ReadDiscoversSecrets(ctx context.Context, c client.Client, logger *logrus.Entry, o client.Object, discoversRef []*discovercrd.DiscoverRef) (secrets []*corev1.Secret, res *reconcile.Result, err error) {
	discoverObjects := make([]object.DiscoverObject, 0, len(discoversRef))
	secrets = make([]*corev1.Secret, 0, len(discoversRef)*2)
	var (
		discoverKafka         *discovercrd.Kafka
		discoverLogstash      *discovercrd.Logstash
		discoverElasticsearch *discovercrd.Elasticsearch
		s                     *corev1.Secret
	)

	// Read discover refs
	for _, ref := range discoversRef {
		if ref == nil {
			continue
		}
		if ref.Kafka != nil {
			discoverKafka = &discovercrd.Kafka{}
			if err = c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: ref.Kafka.Name}, discoverKafka); err != nil {
				if !k8serrors.IsNotFound(err) {
					return nil, nil, errors.Wrapf(err, "Error when read discover Kafka %s", ref.Kafka.Name)
				}
				logger.Warnf("Discover Kafka %s not yet exist, try again later", ref.Kafka.Name)
				return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			discoverObjects = append(discoverObjects, discoverKafka)
		} else if ref.Logstash != nil {
			discoverLogstash = &discovercrd.Logstash{}
			if err = c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: ref.Logstash.Name}, discoverLogstash); err != nil {
				if !k8serrors.IsNotFound(err) {
					return nil, nil, errors.Wrapf(err, "Error when read discover Logstash %s", ref.Logstash.Name)
				}
				logger.Warnf("Discover Logstash %s not yet exist, try again later", ref.Logstash.Name)
				return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			discoverObjects = append(discoverObjects, discoverLogstash)
		} else if ref.Elasticsearch != nil {
			discoverElasticsearch = &discovercrd.Elasticsearch{}
			if err = c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: ref.Elasticsearch.Name}, discoverElasticsearch); err != nil {
				if !k8serrors.IsNotFound(err) {
					return nil, nil, errors.Wrapf(err, "Error when read discover Elasticsearch %s", ref.Elasticsearch.Name)
				}
				logger.Warnf("Discover Elasticsearch %s not yet exist, try again later", ref.Elasticsearch.Name)
				return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			discoverObjects = append(discoverObjects, discoverElasticsearch)
		}
	}

	for _, do := range discoverObjects {
		if do.GetDiscoverStatus().GetSecretEnvRef() == nil || do.GetDiscoverStatus().GetSecretFileRef() == nil {
			logger.Warnf("Discover %s/%s is not ready, try again later", do.GetNamespace(), do.GetName())
			return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Add env secret
		s = &corev1.Secret{}
		if err = c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: *do.GetDiscoverStatus().GetSecretEnvRef()}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return nil, nil, errors.Wrapf(err, "Error when read secret %s", *do.GetDiscoverStatus().GetSecretEnvRef())
			}
			logger.Warnf("Secret %s not yet exist, try again later", *do.GetDiscoverStatus().GetSecretEnvRef())
			return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
		secrets = append(secrets, s)

		// Add file secret
		s = &corev1.Secret{}
		if err = c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: *do.GetDiscoverStatus().GetSecretFileRef()}, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return nil, nil, errors.Wrapf(err, "Error when read secret %s", *do.GetDiscoverStatus().GetSecretFileRef())
			}
			logger.Warnf("Secret %s not yet exist, try again later", *do.GetDiscoverStatus().GetSecretFileRef())
			return nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
		secrets = append(secrets, s)
	}

	return secrets, nil, nil
}

// ReadDiscoverOutputSecrets permit to read the env and file secrets from the discover used as output
func ReadDiscoverOutputSecrets(ctx context.Context, c client.Client, logger *logrus.Entry, o client.Object, discoverOutput *discovercrd.DiscoverRef) (discoverType discovercrd.DiscoverType, secretEnv *corev1.Secret, secretFile *corev1.Secret, res *reconcile.Result, err error) {
	discoverType = discoverOutput.GetType()
	if discoverType == "" {
		return discoverType, nil, nil, nil, nil
	}

	// Compute the label that identify the discover type
	var label string
	switch discoverType {
	case discovercrd.DiscoverTypeKafka:
		label = discovercrd.KafkaAnnotationKey
	case discovercrd.DiscoverTypeElasticsearch:
		label = discovercrd.ElasticsearchAnnotationKey
	case discovercrd.DiscoverTypeLogstash:
		label = discovercrd.LogstashAnnotationKey
	default:
		return discoverType, nil, nil, nil, nil
	}

	selector := fmt.Sprintf("discoverName=%s,%s=true", discoverOutput.GetName(), label)
	labelSelectors, err := labels.Parse(selector)
	if err != nil {
		return discoverType, nil, nil, nil, errors.Wrap(err, "Error when generate label selector")
	}
	secretList := &corev1.SecretList{}
	if err = c.List(ctx, secretList, &client.ListOptions{Namespace: o.GetNamespace(), LabelSelector: labelSelectors}); err != nil {
		return discoverType, nil, nil, nil, errors.Wrap(err, "Error when read discover output secrets")
	}

	logger.Infof("Discover output secrets found: %d with '%s'", len(secretList.Items), selector)

	for _, s := range helper.ToSlicePtr(secretList.Items) {
		if IsDiscoverSecretEnv(s) {
			secretEnv = s
		} else if IsDiscoverSecretFile(s) {
			secretFile = s
		}
	}

	if secretEnv == nil || secretFile == nil {
		logger.Warnf("Discover output secrets for %s not yet exist, try again later", discoverOutput.GetName())
		return discoverType, nil, nil, &reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return discoverType, secretEnv, secretFile, nil, nil
}

// ComputeDiscoverPod injects discover secrets into pod template
func ComputeDiscoverPod(podTemplate k8sbuilder.PodTemplateBuilder, container k8sbuilder.ContainerBuilder, secrets []*corev1.Secret, mountBasePath string) {
	// Compute volumes and volume mounts
	for _, s := range secrets {
		if IsDiscoverSecretFile(s) {
			podTemplate.WithVolumes([]corev1.Volume{
				{
					Name: s.Name,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: s.Name,
						},
					},
				},
			}, k8sbuilder.Merge)

			container.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      s.Name,
					MountPath: fmt.Sprintf("%s/discover/%s", mountBasePath, GetDiscoverMountPathFromAnnotations(s)),
				},
			}, k8sbuilder.Merge)
		} else if IsDiscoverSecretEnv(s) {
			// Compute env from secrets
			container.WithEnvFrom([]corev1.EnvFromSource{
				{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: s.Name,
						},
					},
				},
			}, k8sbuilder.Merge)
		}
	}
}
