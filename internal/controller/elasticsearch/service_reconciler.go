package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ServiceCondition shared.ConditionName = "ServiceReady"
	ServicePhase     shared.PhaseName     = "Service"
)

type serviceReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Service]
}

func newServiceReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Service]) {
	return &serviceReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *corev1.Service](
			client,
			ServicePhase,
			ServiceCondition,
			recorder,
		),
	}
}

// Read existing services
func (r *serviceReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Service], res reconcile.Result, err error) {
	serviceList := &corev1.ServiceList{}
	read = multiphase.NewMultiPhaseRead[*corev1.Service]()

	// Read current node group services
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true,%s/service=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, serviceList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read service")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(serviceList.Items))

	// Generate expected node group services
	expectedServices, err := buildServices(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate services")
	}
	read.SetExpectedObjects(expectedServices)

	return read, res, nil
}
