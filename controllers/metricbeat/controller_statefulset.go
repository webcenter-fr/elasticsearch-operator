package metricbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
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

const (
	StatefulsetCondition = "StatefulsetReady"
	StatefulsetPhase     = "Statefulset"
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
	o := resource.(*beatcrd.Metricbeat)

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
	o := resource.(*beatcrd.Metricbeat)
	sts := &appv1.StatefulSet{}
	s := &corev1.Secret{}
	cm := &corev1.ConfigMap{}
	cmList := &corev1.ConfigMapList{}
	configMapsChecksum := make([]corev1.ConfigMap, 0)
	secretsChecksum := make([]corev1.Secret, 0)

	var (
		es *elasticsearchcrd.Elasticsearch
	)

	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetStatefulsetName(o)}, sts); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read statefulset")
		}
		sts = nil
	}

	data["currentObject"] = sts

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {
		es, err = common.GetElasticsearchFromRef(ctx, r.Client, o, o.Spec.ElasticsearchRef)
		if err != nil {
			return res, errors.Wrap(err, "Error when read ElasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		es = nil
	}

	// Read Custom CA Elasticsearch to generate checksum
	if (o.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled()) || o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		if o.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() && es.IsSelfManagedSecretForTlsApi() {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", GetSecretNameForCAElasticsearch(o))
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
		} else {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			if len(s.Data["ca.crt"]) == 0 {
				return res, errors.Errorf("Secret %s must have a key `ca.crt`", s.Name)
			}

			secretsChecksum = append(secretsChecksum, *s)
		}
	}

	// Read configMaps to generate checksum
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, beatcrd.MetricbeatAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read configMap")
	}
	configMapsChecksum = append(configMapsChecksum, cmList.Items...)

	// Read extra volumes to generate checksum if secret or configmap
	for _, v := range o.Spec.Deployment.AdditionalVolumes {
		if v.ConfigMap != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.ConfigMap.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", v.ConfigMap.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", v.ConfigMap.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}

		if v.Secret != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: v.Secret.SecretName}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", v.Secret.SecretName)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", v.Secret.SecretName)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}
	}

	// Read extra Env to generate checksum if secret or configmap
	for _, env := range o.Spec.Deployment.Env {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.SecretKeyRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", env.ValueFrom.SecretKeyRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", env.ValueFrom.SecretKeyRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: env.ValueFrom.ConfigMapKeyRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", env.ValueFrom.ConfigMapKeyRef.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", env.ValueFrom.ConfigMapKeyRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Read extra Env from to generate checksum if secret or configmap
	for _, ef := range o.Spec.Deployment.EnvFrom {
		if ef.SecretRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.SecretRef.Name}, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", ef.SecretRef.Name)
				}
				r.Log.Warnf("Secret %s not yet exist, try again later", ef.SecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			secretsChecksum = append(secretsChecksum, *s)
			break
		}

		if ef.ConfigMapRef != nil {
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: ef.ConfigMapRef.Name}, cm); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read configMap %s", ef.ConfigMapRef.Name)
				}
				r.Log.Warnf("ConfigMap %s not yet exist, try again later", ef.ConfigMapRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			configMapsChecksum = append(configMapsChecksum, *cm)
			break
		}
	}

	// Generate expected statefulset
	expectedSts, err := BuildStatefulset(o, es, secretsChecksum, configMapsChecksum)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate statefulset")
	}
	data["expectedObject"] = expectedSts

	return res, nil
}

// Diff permit to check if statefulset is up to date
func (r *StatefulsetReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data, patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus())
}

// OnError permit to set status condition on the right state and record error
func (r *StatefulsetReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*beatcrd.Metricbeat)

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
	o := resource.(*beatcrd.Metricbeat)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Statefulset successfully updated")
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

	return res, nil
}
