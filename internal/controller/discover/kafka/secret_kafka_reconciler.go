package kafka

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	strimzicrd "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	secretKafkaCondition shared.ConditionName = "SecretKafkaReady"
	secretKafkaPhase     shared.PhaseName     = "SecretKafka"
)

type secretKafkaReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Kafka, *corev1.Secret]
	isStrimzi bool
}

func newSecretKafkaReconciler(client client.Client, recorder record.EventRecorder, isStrimzi bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Kafka, *corev1.Secret]) {
	return &secretKafkaReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*discovercrd.Kafka, *corev1.Secret](
			client,
			secretKafkaPhase,
			secretKafkaCondition,
			recorder,
			common.FieldManager,
		),
		isStrimzi: isStrimzi,
	}
}

// Read existing Kafka secret
func (r *secretKafkaReconciler) Read(ctx context.Context, o *discovercrd.Kafka, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
	secretList := &corev1.SecretList{}
	var (
		kafkaCluster        *strimzicrd.Kafka
		kafkaCaSecret       *corev1.Secret
		kafkaUserSecret     *corev1.Secret
		kafkaCustomCaSecret *corev1.Secret
	)

	// Read current secrets
	labelSelectors, err := labels.Parse(fmt.Sprintf("discoverName=%s,%s=true", o.Name, discovercrd.KafkaAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, secretList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read secrets")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(secretList.Items))

	// Read Managed Kafka
	if o.Spec.KafkaRef.IsManaged() {

		// Check kube has Strimzi CRD before read
		if !r.isStrimzi {
			return read, res, errors.New("You can't use Kafka managed if Strimzi operator is not installed on your cluster")
		}

		// Read kafka cluster
		kafkaCluster = &strimzicrd.Kafka{}
		namespace := o.Namespace
		if o.Spec.KafkaRef.ManagedKafkaRef.Namespace != "" {
			namespace = o.Spec.KafkaRef.ManagedKafkaRef.Namespace
		}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: namespace, Name: o.Spec.KafkaRef.ManagedKafkaRef.Name}, kafkaCluster); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read kafka %s", o.Spec.KafkaRef.ManagedKafkaRef.Name)
			}
			logger.Warnf("Kafka %s not yet exist, try again later", o.Spec.KafkaRef.ManagedKafkaRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
		if kafkaCluster.Status == nil || len(kafkaCluster.Status.Listeners) == 0 {
			logger.Warnf("Kafka %s not yet ready, try again later", o.Spec.KafkaRef.ManagedKafkaRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Get the CA secret
		if kafkaHaveClusterCa(kafkaCluster) {
			kafkaCaSecret = &corev1.Secret{}
			name := fmt.Sprintf("%s-cluster-ca-cert", kafkaCluster.Name)
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: kafkaCluster.Namespace, Name: name}, kafkaCaSecret); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", name)
				}
				logger.Warnf("Secret not found %s/%s, try latter", kafkaCluster.Namespace, name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Set secret ref in status
			data["kafkaCASecretRef"] = ptr.To(name)
		}

		// Get the Kafka user
		if o.Spec.KafkaRef.ManagedKafkaRef.UserRef != nil && o.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name != "" {
			kafkaUser := &strimzicrd.KafkaUser{}
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: namespace, Name: o.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name}, kafkaUser); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read kafka user %s", o.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name)
				}
				logger.Warnf("Kafka user not found %s/%s, try latter", namespace, o.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			if kafkaUser.Status == nil || kafkaUser.Status.Secret == nil {
				logger.Warnf("Kafka user %s/%s not ready, try latter", namespace, o.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Set secret ref in status
			data["kafkaUserSecretRef"] = kafkaUser.Status.Secret

			// Read secret that store user certificates
			kafkaUserSecret = &corev1.Secret{}
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: kafkaUser.Namespace, Name: *kafkaUser.Status.Secret}, kafkaUserSecret); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", *kafkaUser.Status.Secret)
				}
				logger.Warnf("Secret not found %s/%s, try latter", kafkaUser.Namespace, *kafkaUser.Status.Secret)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}

	}

	// Read external kafka user secret
	if o.Spec.KafkaRef.IsExternal() && o.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef != nil && o.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef.Name != "" {
		kafkaUserSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef.Name}, kafkaUserSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Set secret ref in status
		data["kafkaUserSecretRef"] = ptr.To(kafkaUserSecret.Name)
	}

	// Read custom CA secret
	if o.Spec.KafkaRef.KafkaCaSecretRef != nil && o.Spec.KafkaRef.KafkaCaSecretRef.Name != "" {
		kafkaCustomCaSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.KafkaRef.KafkaCaSecretRef.Name}, kafkaCustomCaSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.KafkaRef.KafkaCaSecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.KafkaRef.KafkaCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Build expected secret
	expectedSecretsKafka, err := buildKafkaSecrets(o, kafkaCluster, kafkaCaSecret, kafkaUserSecret, kafkaCustomCaSecret)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when build expected secret")
	}

	preserveKafkaGeneratedKeystores(o, read.GetCurrentObjects(), expectedSecretsKafka)

	common.InjectTypeMeta(r.Client().Scheme(), expectedSecretsKafka...)
	read.SetExpectedObjects(expectedSecretsKafka)

	return read, res, nil
}

// preserveKafkaGeneratedKeystores reuses existing generated PKCS#12 keystores/truststores
// and passwords when the underlying certificates have not changed, avoiding unnecessary
// secret mutations and pod rollouts on every SSA apply loop.
func preserveKafkaGeneratedKeystores(o *discovercrd.Kafka, currentSecrets, expectedSecrets []*corev1.Secret) {
	envSuffix := getEnvSuffix(o)
	userPassKey := fmt.Sprintf("KAFKA_USER_PASSWORD_%s", envSuffix)
	caPassKey := fmt.Sprintf("KAFKA_CA_TRUSTSTORE_PASSWORD_%s", envSuffix)

	currentByName := make(map[string]*corev1.Secret, len(currentSecrets))
	for _, s := range currentSecrets {
		currentByName[s.Name] = s
	}

	expectedEnv := currentByName[GetSecretNameForEnv(o)]
	expectedFile := currentByName[GetSecretNameForFile(o)]
	if expectedEnv == nil || expectedFile == nil {
		return
	}

	var newEnv, newFile *corev1.Secret
	for _, s := range expectedSecrets {
		switch s.Name {
		case GetSecretNameForEnv(o):
			newEnv = s
		case GetSecretNameForFile(o):
			newFile = s
		}
	}
	if newEnv == nil || newFile == nil {
		return
	}

	// Preserve user.p12 + password if user.crt and user.key are unchanged
	if len(newFile.Data["user.p12"]) > 0 &&
		len(expectedFile.Data["user.p12"]) > 0 &&
		len(expectedEnv.Data[userPassKey]) > 0 &&
		bytes.Equal(newFile.Data["user.crt"], expectedFile.Data["user.crt"]) &&
		bytes.Equal(newFile.Data["user.key"], expectedFile.Data["user.key"]) {
		newFile.Data["user.p12"] = expectedFile.Data["user.p12"]
		newEnv.Data[userPassKey] = expectedEnv.Data[userPassKey]
	}

	// Preserve ca.p12 + password if ca.crt is unchanged
	if len(newFile.Data["ca.p12"]) > 0 &&
		len(expectedFile.Data["ca.p12"]) > 0 &&
		len(expectedEnv.Data[caPassKey]) > 0 &&
		bytes.Equal(newFile.Data["ca.crt"], expectedFile.Data["ca.crt"]) {
		newFile.Data["ca.p12"] = expectedFile.Data["ca.p12"]
		newEnv.Data[caPassKey] = expectedEnv.Data[caPassKey]
	}
}
