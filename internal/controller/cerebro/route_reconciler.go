package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RouteCondition shared.ConditionName = "RouteReady"
	RoutePhase     shared.PhaseName     = "Route"
)

type routeReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *routev1.Route]
}

func newRouteReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *routev1.Route]) {
	return &routeReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *routev1.Route](
			client,
			RoutePhase,
			RouteCondition,
			recorder,
		),
	}
}

// Read existing route
func (r *routeReconciler) Read(ctx context.Context, o *cerebrocrd.Cerebro, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*routev1.Route], res reconcile.Result, err error) {
	route := &routev1.Route{}
	read = multiphase.NewMultiPhaseRead[*routev1.Route]()

	// Read current route
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, route); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read route")
		}
		route = nil
	}
	if route != nil {
		read.AddCurrentObject(route)
	}

	// Generate expected route
	expectedRoutes, err := buildRoutes(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate route")
	}
	read.SetExpectedObjects(expectedRoutes)

	return read, res, nil
}
