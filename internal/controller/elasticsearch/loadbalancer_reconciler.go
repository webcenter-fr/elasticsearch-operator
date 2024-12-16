package elasticsearch

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LoadBalancerCondition shared.ConditionName = "LoadBalancerReady"
	LoadBalancerPhase     shared.PhaseName     = "LoadBalancer"
)

type loadBalancerReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newLoadBalancerReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &loadBalancerReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			LoadBalancerPhase,
			LoadBalancerCondition,
			recorder,
		),
	}
}

// Read existing load balancer
func (r *loadBalancerReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	lb := &corev1.Service{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current load balancer
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLoadBalancerName(o)}, lb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read load balancer")
		}
		lb = nil
	}
	if lb != nil {
		read.SetCurrentObjects([]client.Object{lb})
	}

	// Generate expected load balancer
	expectedLbs, err := buildLoadbalancers(o)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate load balancer")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedLbs))

	return read, res, nil
}
