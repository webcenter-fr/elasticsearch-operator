package cerebro

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ConfigmapCondition shared.ConditionName = "ConfigmapReady"
	ConfigmapPhase     shared.PhaseName     = "Configmap"
)

type configMapReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.ConfigMap]
}

func newConfiMapReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.ConfigMap]) {
	return &configMapReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.ConfigMap](
			client,
			ConfigmapPhase,
			ConfigmapCondition,
			recorder,
		),
	}
}

// Read existing configmaps
func (r *configMapReconciler) Read(ctx context.Context, o *cerebrocrd.Cerebro, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.ConfigMap], res reconcile.Result, err error) {
	cm := &corev1.ConfigMap{}
	read = multiphase.NewMultiPhaseRead[*corev1.ConfigMap]()
	hostList := &cerebrocrd.HostList{}
	var es *elasticsearchcrd.Elasticsearch
	esList := make([]elasticsearchcrd.Elasticsearch, 0)
	esExternalList := make([]cerebrocrd.ElasticsearchExternalRef, 0)

	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetConfigMapName(o)}, cm); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrap(err, "Error when read config maps")
		}
		cm = nil
	}
	if cm != nil {
		read.AddCurrentObject(cm)
	}

	// Read Elasticsearch linked to cerebro
	// Add and clean finalizer to track change on Host because of there are not controller on it
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.cerebroRef.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err = r.Client().List(ctx, hostList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return read, res, errors.Wrap(err, "error when read Cerebro hosts")
	}
	for _, host := range hostList.Items {
		// Handle finalizer
		if !host.DeletionTimestamp.IsZero() {
			controllerutil.RemoveFinalizer(&host, finalizer.String())
			if err = r.Client().Update(ctx, &host); err != nil {
				return read, res, errors.Wrapf(err, "Error when delete finalizer on Host %s", host.Name)
			}
			logger.Debugf("Remove finalizer on Cerebro host %s/%s", host.Namespace, host.Name)

			// Remove Elasticsearch finalizer if cluster is managed and no more exist
			if host.Spec.ElasticsearchRef.IsManaged() {
				es = &elasticsearchcrd.Elasticsearch{}
				if err = r.Client().Get(ctx, types.NamespacedName{Namespace: host.Namespace, Name: host.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}, es); err != nil {
					if k8serrors.IsNotFound(err) {
						controllerutil.RemoveFinalizer(&host, finalizer.String())
						if err = r.Client().Update(ctx, &host); err != nil {
							return read, res, errors.Wrapf(err, "Error when delete finalizer on Host %s", host.Name)
						}
						logger.Debugf("Remove finalizer on Cerebro host %s/%s", host.Namespace, host.Name)
					}
				}
			}

			continue
		}
		if !controllerutil.ContainsFinalizer(&host, finalizer.String()) {
			controllerutil.AddFinalizer(&host, finalizer.String())
			if err = r.Client().Update(ctx, &host); err != nil {
				return read, res, errors.Wrapf(err, "Error when add finalizer on Host %s", host.Name)
			}
		}

		if host.Spec.ElasticsearchRef.IsManaged() {
			es = &elasticsearchcrd.Elasticsearch{}
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: host.Namespace, Name: host.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}, es); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrap(err, "Error when read elasticsearch")
				}
			} else {
				esList = append(esList, *es)
			}
		} else if host.Spec.ElasticsearchRef.IsExternal() {
			esExternalList = append(esExternalList, *host.Spec.ElasticsearchRef.ExternalElasticsearchRef)
		}

	}

	// Generate expected node group configmaps
	expectedCms, err := buildConfigMaps(o, esList, esExternalList)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate config maps")
	}
	read.SetExpectedObjects(expectedCms)

	return read, res, nil
}
