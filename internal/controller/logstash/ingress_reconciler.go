package logstash

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	IngressCondition shared.ConditionName = "IngressReady"
	IngressPhase     shared.PhaseName     = "Ingress"
)

type ingressReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *networkingv1.Ingress]
}

func newIngressReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *networkingv1.Ingress]) {
	return &ingressReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *networkingv1.Ingress](
			client,
			IngressPhase,
			IngressCondition,
			recorder,
		),
	}
}

// Read existing ingress
func (r *ingressReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*networkingv1.Ingress], res reconcile.Result, err error) {
	ingressList := &networkingv1.IngressList{}
	read = multiphase.NewMultiPhaseRead[*networkingv1.Ingress]()

	// Read current ingress
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, logstashcrd.LogstashAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, ingressList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read ingress")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(ingressList.Items))

	// Generate expected ingress
	expectedIngresses, err := buildIngresses(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate ingress")
	}
	read.SetExpectedObjects(expectedIngresses)

	return read, res, nil
}
