package kibana

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.ConfigMap]
}

func newConfiMapReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.ConfigMap]) {
	return &configMapReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.ConfigMap](
			client,
			ConfigmapPhase,
			ConfigmapCondition,
			recorder,
		),
	}
}

// Read existing configmaps
func (r *configMapReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.ConfigMap], res reconcile.Result, err error) {
	cm := &corev1.ConfigMap{}
	read = multiphase.NewMultiPhaseRead[*corev1.ConfigMap]()
	var es *elasticsearchcrd.Elasticsearch

	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetConfigMapName(o)}, cm); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read config maps")
		}
		cm = nil
	}
	if cm != nil {
		read.AddCurrentObject(cm)
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

	// Generate expected node group configmaps
	expectedCms, err := buildConfigMaps(o, es)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate config maps")
	}
	read.SetExpectedObjects(expectedCms)

	return read, res, nil
}
