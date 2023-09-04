package kibana

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LoadBalancerCondition common.ConditionName = "LoadBalancerReady"
	LoadBalancerPhase     common.PhaseName     = "LoadBalancer"
)

type LoadBalancerReconciler struct {
	common.Reconciler
}

func NewLoadBalancerReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &LoadBalancerReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "loadBalancer",
			}),
			Name:   "loadBalancer",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *LoadBalancerReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, LoadBalancerCondition, LoadBalancerPhase)
}

// Read existing load balancer
func (r *LoadBalancerReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*kibanacrd.Kibana)
	lb := &corev1.Service{}

	// Read current load balancer
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetLoadBalancerName(o)}, lb); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read load balancer")
		}
		lb = nil
	}
	data["currentObject"] = lb

	// Generate expected load balancer
	expectedLb, err := BuildLoadbalancer(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate load balancer")
	}
	data["expectedObject"] = expectedLb

	return res, nil
}

// Diff permit to check if load balancer is up to date
func (r *LoadBalancerReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *LoadBalancerReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, LoadBalancerCondition, LoadBalancerPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LoadBalancerReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, LoadBalancerCondition, LoadBalancerPhase)
}
