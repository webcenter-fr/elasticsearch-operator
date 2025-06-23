package filebeat

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ConfigmapCondition shared.ConditionName = "ConfigmapReady"
	ConfigmapPhase     shared.PhaseName     = "Configmap"
)

type configMapReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ConfigMap]
}

func newConfiMapReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ConfigMap]) {
	return &configMapReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*beatcrd.Filebeat, *corev1.ConfigMap](
			client,
			ConfigmapPhase,
			ConfigmapCondition,
			recorder,
		),
	}
}

// Read existing configmaps
func (r *configMapReconciler) Read(ctx context.Context, o *beatcrd.Filebeat, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.ConfigMap], res reconcile.Result, err error) {
	cmList := &corev1.ConfigMapList{}
	var (
		es                    *elasticsearchcrd.Elasticsearch
		ls                    *logstashcrd.Logstash
		elasticsearchCASecret *corev1.Secret
		logstashCASecret      *corev1.Secret
	)
	read = multiphase.NewMultiPhaseRead[*corev1.ConfigMap]()

	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, beatcrd.FilebeatAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(cmList.Items))

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef != nil && o.Spec.ElasticsearchRef.IsManaged() {
		es, err = common.GetElasticsearchFromRef(ctx, r.Client(), o, *o.Spec.ElasticsearchRef)
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
	if o.Spec.LogstashRef != nil && o.Spec.LogstashRef.IsManaged() {
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

	} else {
		ls = nil
	}

	// Read ElasticsearchCASecret
	if o.Spec.ElasticsearchRef != nil && (o.Spec.ElasticsearchRef.IsExternal() || o.Spec.ElasticsearchRef.IsManaged()) && o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		elasticsearchCASecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}, elasticsearchCASecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read elasticsearchCASecret %s", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
			}
			logger.Warnf("elasticsearchCASecret %s not yet exist, try again later", o.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Read logstashCASecret
	if o.Spec.LogstashRef != nil && (o.Spec.LogstashRef.IsExternal() || o.Spec.LogstashRef.IsManaged()) && o.Spec.LogstashRef.LogstashCaSecretRef != nil {
		logstashCASecret = &corev1.Secret{}
		if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LogstashRef.LogstashCaSecretRef.Name}, logstashCASecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read logstashCASecret %s", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			}
			logger.Warnf("logstashCASecret %s not yet exist, try again later", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Generate expected node group configmaps
	expectedCms, err := buildConfigMaps(o, es, ls, elasticsearchCASecret, logstashCASecret)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate config maps")
	}
	read.SetExpectedObjects(expectedCms)

	return read, res, nil
}
