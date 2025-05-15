package logstash

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RouteCondition shared.ConditionName = "RouteReady"
	RoutePhase     shared.PhaseName     = "Route"
)

type routeReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *routev1.Route]
}

func newRouteReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *routev1.Route]) {
	return &routeReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*logstashcrd.Logstash, *routev1.Route](
			client,
			RoutePhase,
			RouteCondition,
			recorder,
		),
	}
}

// Read existing route
func (r *routeReconciler) Read(ctx context.Context, o *logstashcrd.Logstash, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*routev1.Route], res reconcile.Result, err error) {
	routeList := &routev1.RouteList{}
	read = multiphase.NewMultiPhaseRead[*routev1.Route]()

	// Read current route
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, logstashcrd.LogstashAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, routeList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read route")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(routeList.Items))

	// Generate expected route
	expectedRoutes, err := buildRoutes(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate route")
	}
	read.SetExpectedObjects(expectedRoutes)

	return read, res, nil
}
