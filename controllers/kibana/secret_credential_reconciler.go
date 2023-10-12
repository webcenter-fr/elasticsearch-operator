package kibana

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
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
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
	CredentialCondition shared.ConditionName = "CredentialReady"
	CredentialPhase     shared.PhaseName     = "Credential"
)

type credentialReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newCredentialReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &credentialReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			CredentialPhase,
			CredentialCondition,
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
func (r *credentialReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	s := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()
	sEs := &corev1.Secret{}

	var es *elasticsearchcrd.Elasticsearch

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
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

		// Read secret that store Elasticsearch crdentials
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForCredentials(es)}, sEs); err != nil {
			if !k8serrors.IsNotFound(err) {
				return read, res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForCredentials(es))
			}
			r.Log.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForCredentials(es))
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Generate expected secret
	expectedSecretCredentials, err := buildCredentialSecrets(o, sEs)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCredentials(o))
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedSecretCredentials))

	return read, res, nil
}
