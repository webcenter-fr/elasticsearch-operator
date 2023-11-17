package filebeat

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigmapCondition shared.ConditionName = "ConfigmapReady"
	ConfigmapPhase     shared.PhaseName     = "Configmap"
)

type configMapReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newConfiMapReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &configMapReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			ConfigmapPhase,
			ConfigmapCondition,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
	}
}

// Read existing configmaps
func (r *configMapReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)
	cmList := &corev1.ConfigMapList{}
	var (
		es               *elasticsearchcrd.Elasticsearch
		logstashCASecret *corev1.Secret
	)
	read = controller.NewBasicMultiPhaseRead()

	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, beatcrd.FilebeatAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, cmList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read configMap")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(cmList.Items))

	// Read Elasticsearch
	if o.Spec.ElasticsearchRef.IsManaged() {
		es, err = common.GetElasticsearchFromRef(ctx, r.Client, o, o.Spec.ElasticsearchRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read ElasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		es = nil
	}

	// Read logstashCASecret
	if (o.Spec.LogstashRef.IsExternal() || o.Spec.LogstashRef.IsManaged()) && o.Spec.LogstashRef.LogstashCaSecretRef != nil {
		logstashCASecret = &corev1.Secret{}
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.LogstashRef.LogstashCaSecretRef.Name}, logstashCASecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read logstashCASecret %s", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			}
			r.Log.Warnf("logstashCASecret %s not yet exist, try again later", o.Spec.LogstashRef.LogstashCaSecretRef.Name)
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Generate expected node group configmaps
	expectedCms, err := buildConfigMaps(o, es, logstashCASecret)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate config maps")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedCms))

	return read, res, nil
}
