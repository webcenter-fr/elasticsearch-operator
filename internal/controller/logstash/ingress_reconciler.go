package logstash

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IngressCondition shared.ConditionName = "IngressReady"
	IngressPhase     shared.PhaseName     = "Ingress"
)

type ingressReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newIngressReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &ingressReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			IngressPhase,
			IngressCondition,
			recorder,
		),
	}
}

// Read existing ingress
func (r *ingressReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*logstashcrd.Logstash)
	ingressList := &networkingv1.IngressList{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current ingress
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, logstashcrd.LogstashAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, ingressList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read ingress")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(ingressList.Items))

	// Generate expected ingress
	expectedIngresses, err := buildIngresses(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate ingress")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedIngresses))

	return read, res, nil
}
