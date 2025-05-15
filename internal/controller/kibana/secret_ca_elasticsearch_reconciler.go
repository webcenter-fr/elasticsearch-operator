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
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	CAElasticsearchCondition shared.ConditionName = "CAElasticsearchReady"
	CAElasticsearchPhase     shared.PhaseName     = "CAElasticsearch"
)

type caElasticsearchReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.Secret]
}

func newCAElasticsearchReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.Secret]) {
	return &caElasticsearchReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *corev1.Secret](
			client,
			CAElasticsearchPhase,
			CAElasticsearchCondition,
			recorder,
		),
	}
}

// Read existing secret
func (r *caElasticsearchReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Secret], res reconcile.Result, err error) {
	s := &corev1.Secret{}
	read = multiphase.NewMultiPhaseRead[*corev1.Secret]()
	sEs := &corev1.Secret{}

	var es *elasticsearchcrd.Elasticsearch

	// Read current secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
		}
		s = nil
	}
	if s != nil {
		read.AddCurrentObject(s)
	}

	if o.Spec.ElasticsearchRef.IsManaged() {
		// Read Elasticsearch
		es, err = common.GetElasticsearchFromRef(ctx, r.Client(), o, o.Spec.ElasticsearchRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read elasticsearchRef")
		}
		if es == nil {
			logger.Warn("ElasticsearchRef not found, try latter")
			return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Check if mirror CAApiPKI from Elasticsearch CRD
		if es.Spec.Tls.IsTlsEnabled() {
			// Read secret that store Elasticsearch API certs
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForTlsApi(es)}, sEs); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				}
				logger.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Generate expected secret
	expectedSecretCAElasticsearchs, err := buildCAElasticsearchSecrets(o, sEs)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCAElasticsearch(o))
	}
	read.SetExpectedObjects(expectedSecretCAElasticsearchs)

	return read, res, nil
}
