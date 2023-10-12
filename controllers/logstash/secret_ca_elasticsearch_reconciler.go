package logstash

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CAElasticsearchCondition shared.ConditionName = "CAElasticsearchReady"
	CAElasticsearchPhase     shared.PhaseName     = "CAElasticsearch"
)

type caElasticsearchReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newCAElasticsearchReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &caElasticsearchReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			CAElasticsearchPhase,
			CAElasticsearchCondition,
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

// Read existing secret
func (r *caElasticsearchReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {

	o := resource.(*logstashcrd.Logstash)
	s := &corev1.Secret{}
	sEs := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	var es *elasticsearchcrd.Elasticsearch

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
		}
		s = nil
	}
	if s != nil {
		read.SetCurrentObjects([]client.Object{s})
	}

	if o.Spec.ElasticsearchRef.IsManaged() {
		// Read Elasticsearch
		es, err = common.GetElasticsearchFromRef(ctx, r.Client, o, o.Spec.ElasticsearchRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read elasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Check if mirror CAApiPKI from Elasticsearch CRD
		if es.IsTlsApiEnabled() {
			// Read secret that store Elasticsearch API certs
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForTlsApi(es)}, sEs); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				}
				r.Log.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Generate expected secret
	expectedSecretCAElasticsearchs, err := buildCAElasticsearchSecrets(o, sEs)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCAElasticsearch(o))
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedSecretCAElasticsearchs))

	return read, res, nil
}
