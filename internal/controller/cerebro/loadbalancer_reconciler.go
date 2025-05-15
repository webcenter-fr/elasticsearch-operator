package cerebro

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	LoadBalancerCondition shared.ConditionName = "LoadBalancerReady"
	LoadBalancerPhase     shared.PhaseName     = "LoadBalancer"
)

type loadBalancerReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service]
}

func newLoadBalancerReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service]) {
	return &loadBalancerReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*cerebrocrd.Cerebro, *corev1.Service](
			client,
			LoadBalancerPhase,
			LoadBalancerCondition,
			recorder,
		),
	}
}

// Read existing load balancer
func (r *loadBalancerReconciler) Read(ctx context.Context, o *cerebrocrd.Cerebro, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*corev1.Service], res reconcile.Result, err error) {
	lb := &corev1.Service{}
	read = multiphase.NewMultiPhaseRead[*corev1.Service]()

	// Read current load balancer
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLoadBalancerName(o)}, lb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read load balancer")
		}
		lb = nil
	}
	if lb != nil {
		read.AddCurrentObject(lb)
	}

	// Generate expected load balancer
	expectedLbs, err := buildLoadbalancers(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate load balancer")
	}
	read.SetExpectedObjects(expectedLbs)

	return read, res, nil
}
