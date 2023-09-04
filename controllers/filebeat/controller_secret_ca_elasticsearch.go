package filebeat

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CAElasticsearchCondition common.ConditionName = "CAElasticsearchReady"
	CAElasticsearchPhase     common.PhaseName     = "CAElasticsearch"
)

type CAElasticsearchReconciler struct {
	common.Reconciler
}

func NewCAElasticsearchReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &CAElasticsearchReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "caElasticsearch",
			}),
			Name:   "caElasticsearch",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *CAElasticsearchReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, CAElasticsearchCondition, CAElasticsearchPhase)
}

// Read existing secret
func (r *CAElasticsearchReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {

	o := resource.(*beatcrd.Filebeat)
	s := &corev1.Secret{}
	sEs := &corev1.Secret{}

	var es *elasticsearchcrd.Elasticsearch
	var expectedSecretCAElasticsearch *corev1.Secret

	// Read current secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCAElasticsearch(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCAElasticsearch(o))
		}
		s = nil
	}
	data["currentObject"] = s

	if o.Spec.ElasticsearchRef.IsManaged() {

		// Read Elasticsearch
		es, err = common.GetElasticsearchFromRef(ctx, r.Client, o, o.Spec.ElasticsearchRef)
		if err != nil {
			return res, errors.Wrap(err, "Error when read elasticsearchRef")
		}
		if es == nil {
			r.Log.Warn("ElasticsearchRef not found, try latter")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Check if mirror CAApiPKI from Elasticsearch CRD
		if es.IsTlsApiEnabled() {
			// Read secret that store Elasticsearch API certs
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: elasticsearchcontrollers.GetSecretNameForTlsApi(es)}, sEs); err != nil {
				if !k8serrors.IsNotFound(err) {
					return res, errors.Wrapf(err, "Error when read secret %s", elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				}
				r.Log.Warnf("Secret not found %s/%s, try latter", es.Namespace, elasticsearchcontrollers.GetSecretNameForTlsApi(es))
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Generate expected secret
	expectedSecretCAElasticsearch, err = BuildCAElasticsearchSecret(o, sEs)
	if err != nil {
		return res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCAElasticsearch(o))
	}
	data["expectedObject"] = expectedSecretCAElasticsearch

	return res, nil
}

// Diff permit to check if credential secret is up to date
func (r *CAElasticsearchReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)

	// No PkiCA to mirror
	if !o.Spec.ElasticsearchRef.IsManaged() {
		return diff, res, nil
	}

	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *CAElasticsearchReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, CAElasticsearchCondition, CAElasticsearchPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *CAElasticsearchReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, CAElasticsearchCondition, CAElasticsearchPhase)
}
