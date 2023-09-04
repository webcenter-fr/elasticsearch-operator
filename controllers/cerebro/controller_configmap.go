package cerebro

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ConfigmapCondition common.ConditionName = "ConfigmapReady"
	ConfigmapPhase     common.PhaseName     = "Configmap"
)

type ConfigMapReconciler struct {
	common.Reconciler
}

func NewConfiMapReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &ConfigMapReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "configMap",
			}),
			Name:   "configMap",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *ConfigMapReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, ConfigmapCondition, ConfigmapPhase)
}

// Read existing configmaps
func (r *ConfigMapReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*cerebrocrd.Cerebro)
	cm := &corev1.ConfigMap{}
	hostList := &cerebrocrd.HostList{}
	var es *elasticsearchcrd.Elasticsearch
	esList := make([]elasticsearchcrd.Elasticsearch, 0)

	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetConfigMapName(o)}, cm); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrap(err, "Error when read config maps")
		}
		cm = nil
	}

	data["currentObject"] = cm

	// Read Elasticsearch linked to cerebro
	// Add and clean finalizer to track change on Host because of there are not controller on it
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.cerebroRef.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err = r.Client.List(ctx, hostList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrap(err, "error when read Cerebro hosts")
	}
	for _, host := range hostList.Items {
		// Handle finalizer
		if !host.DeletionTimestamp.IsZero() {
			controllerutil.RemoveFinalizer(&host, CerebroFinalizer)
			if err = r.Client.Update(ctx, &host); err != nil {
				return res, errors.Wrapf(err, "Error when add finalizer on Host %s", host.Name)
			}
			continue
		}
		if !controllerutil.ContainsFinalizer(&host, CerebroFinalizer) {
			controllerutil.AddFinalizer(&host, CerebroFinalizer)
			if err = r.Client.Update(ctx, &host); err != nil {
				return res, errors.Wrapf(err, "Error when add finalizer on Host %s", host.Name)
			}
		}

		es = &elasticsearchcrd.Elasticsearch{}
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: host.Namespace, Name: host.Spec.ElasticsearchRef}, es); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrap(err, "Error when read elasticsearch")
			}
		} else {
			esList = append(esList, *es)
		}
	}

	// Generate expected node group configmaps
	expectedCm, err := BuildConfigMap(o, esList)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate config maps")
	}
	data["expectedObject"] = expectedCm

	return res, nil
}

// Diff permit to check if configmaps are up to date
func (r *ConfigMapReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *ConfigMapReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, ConfigmapCondition, ConfigmapPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ConfigMapReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, ConfigmapCondition, ConfigmapPhase)
}
