package filebeat

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceCondition shared.ConditionName = "ServiceReady"
	ServicePhase     shared.PhaseName     = "Service"
)

type serviceReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newServiceReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &serviceReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			ServicePhase,
			ServiceCondition,
			recorder,
		),
	}
}

// Read existing services
func (r *serviceReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)
	serviceList := &corev1.ServiceList{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current services
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, beatcrd.FilebeatAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, serviceList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read service")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(serviceList.Items))

	// Generate expected service
	expectedServices, err := buildServices(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate service")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedServices))

	return read, res, nil
}
