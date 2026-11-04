package logstash

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/sirupsen/logrus"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/logstash"
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
	secretLogstashCondition shared.ConditionName = "SecretLogstashReady"
	secretLogstashPhase     shared.PhaseName     = "SecretLogstash"
)

type secretLogstashReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Logstash, *corev1.Secret]
}

func newSecretLogstashReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*discovercrd.Logstash, *corev1.Secret]) {
	return &secretLogstashReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*discovercrd.Logstash, *corev1.Secret](
			client,
			secretLogstashPhase,
			secretLogstashCondition,
			recorder,
			common.FieldManager,
		),
	}
}

// Read existing Logstash secret
func (r *secretLogstashReconciler) Read(ctx context.Context, o *discovercrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
	secretList := &corev1.SecretList{}
	var (
		ls                     *logstashcrd.Logstash
		logstashCaSecret       *corev1.Secret
		logstashUserSecret     *corev1.Secret
		logstashCustomCaSecret *corev1.Secret
	)

	// Read existing secrets
	labelSelectors, err := labels.Parse(fmt.Sprintf("discoverName=%s,%s=true", o.Name, discovercrd.LogstashAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, secretList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read secrets")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(secretList.Items))

	// Read Managed Logstash
	if o.Spec.LogstashRef.IsManaged() {

		// Read logstash cluster
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

		// Get the user secret
		if o.Spec.LogstashRef.ManagedLogstashRef.UserRef != nil && o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Name != "" {
			userNamespace := o.Namespace
			if o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace != "" {
				userNamespace = o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace
			}
			logstashUserSecret = &corev1.Secret{}
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: userNamespace, Name: o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Name}, logstashUserSecret); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Name)
				}
				logger.Warnf("Secret not found %s/%s, try latter", userNamespace, o.Spec.LogstashRef.ManagedLogstashRef.UserRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Set secret ref in status
			data["logstashUserSecretRef"] = ptr.To(logstashUserSecret.Name)
		}

		// Get the CA secret
		if ls.Spec.Pki.IsEnabled() {
			logstashCaSecret = &corev1.Secret{}
			name := logstashcontrollers.GetSecretNameForTls(ls)
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: ls.Namespace, Name: name}, logstashCaSecret); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", name)
				}
				logger.Warnf("Secret not found %s/%s, try latter", ls.Namespace, name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}

			data["logstashCASecretRef"] = ptr.To(logstashCaSecret.Name)
		}
	}

	// Read logstash user secret if external
	if o.Spec.LogstashRef.IsExternal() && o.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef != nil && o.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef.Name != "" {
		logstashUserSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef.Name}, logstashUserSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Read custom CA secret
	if o.Spec.LogstashRef.LogstashCaSecretRef != nil && o.Spec.LogstashRef.LogstashCaSecretRef.Name != "" {
		logstashCustomCaSecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LogstashRef.LogstashCaSecretRef.Name}, logstashCustomCaSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			}
			logger.Warnf("Secret not found %s/%s, try latter", o.Namespace, o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Build expected secret
	expectedSecretsLogstash, err := buildLogstashSecrets(o, ls, logstashCaSecret, logstashUserSecret, logstashCustomCaSecret)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when build expected secret")
	}

	preserveLogstashGeneratedKeystore(o, read.GetCurrentObjects(), expectedSecretsLogstash)

	common.InjectTypeMeta(r.Client().Scheme(), expectedSecretsLogstash...)
	read.SetExpectedObjects(expectedSecretsLogstash)

	return read, res, nil
}

// preserveLogstashGeneratedKeystore reuses the existing generated PKCS#12 keystore and
// password when the underlying user certificate and key have not changed, avoiding
// unnecessary secret mutations and pod rollouts on every SSA apply loop.
func preserveLogstashGeneratedKeystore(o *discovercrd.Logstash, currentSecrets, expectedSecrets []*corev1.Secret) {
	envSuffix := getEnvSuffix(o)
	userPassKey := fmt.Sprintf("LOGSTASH_USER_PASSWORD_%s", envSuffix)

	currentByName := make(map[string]*corev1.Secret, len(currentSecrets))
	for _, s := range currentSecrets {
		currentByName[s.Name] = s
	}

	currentEnv := currentByName[GetSecretNameForEnv(o)]
	currentFile := currentByName[GetSecretNameForFile(o)]
	if currentEnv == nil || currentFile == nil {
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

	if len(newFile.Data["user.p12"]) > 0 &&
		len(currentFile.Data["user.p12"]) > 0 &&
		len(currentEnv.Data[userPassKey]) > 0 &&
		bytes.Equal(newFile.Data["user.crt"], currentFile.Data["user.crt"]) &&
		bytes.Equal(newFile.Data["user.key"], currentFile.Data["user.key"]) {
		newFile.Data["user.p12"] = currentFile.Data["user.p12"]
		newEnv.Data[userPassKey] = currentEnv.Data[userPassKey]
	}
}
