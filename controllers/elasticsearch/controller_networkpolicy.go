package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	NetworkPolicyCondition common.ConditionName = "NetworkPolicyReady"
	NetworkPolicyPhase     common.PhaseName     = "NetworkPolicy"
)

type NetworkPolicyReconciler struct {
	common.Reconciler
}

func NewNetworkPolicyReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &NetworkPolicyReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "networkPolicy",
			}),
			Name:   "networkPolicy",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *NetworkPolicyReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, NetworkPolicyCondition, NetworkPolicyPhase)
}

// Read existing network policy
func (r *NetworkPolicyReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	np := &networkingv1.NetworkPolicy{}
	kibanaList := &kibanacrd.KibanaList{}
	logstashList := &logstashcrd.LogstashList{}
	filebeatList := &beatcrd.FilebeatList{}
	metricbeatList := &beatcrd.MetricbeatList{}
	hostList := &cerebrocrd.HostList{}
	oList := make([]client.Object, 0)
	var cb *cerebrocrd.Cerebro
	var oListTmp []client.Object

	// Read current network policy
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetNetworkPolicyName(o)}, np); err != nil {
		if !k8serrors.IsNotFound(err) {
			return res, errors.Wrapf(err, "Error when read network policy")
		}
		np = nil
	}
	data["currentObject"] = np

	// Read remote target that access on this Elasticsearch cluster
	// Read kibana referer
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client.List(context.Background(), kibanaList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrapf(err, "Error when read Kibana")
	}
	oListTmp = helper.ToSliceOfObject(kibanaList.Items)
	for _, kb := range oListTmp {
		if kb.GetNamespace() != o.Namespace {
			oList = append(oList, kb)
		}
	}

	// Read Logstash referer
	fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client.List(context.Background(), logstashList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrapf(err, "Error when read Logstash")
	}
	oListTmp = helper.ToSliceOfObject(logstashList.Items)
	for _, ls := range oListTmp {
		if ls.GetNamespace() != o.Namespace {
			oList = append(oList, ls)
		}
	}

	// Read filebeat referer
	fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client.List(context.Background(), filebeatList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrapf(err, "Error when read filebeat")
	}
	oListTmp = helper.ToSliceOfObject(filebeatList.Items)
	for _, fb := range oListTmp {
		if fb.GetNamespace() != o.Namespace {
			oList = append(oList, fb)
		}
	}

	// Read metricbeat referer
	fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", o.GetNamespace(), o.GetName()))
	if err := r.Client.List(context.Background(), metricbeatList, &client.ListOptions{FieldSelector: fs}); err != nil {
		return res, errors.Wrapf(err, "Error when read metricbeat")
	}
	oListTmp = helper.ToSliceOfObject(metricbeatList.Items)
	for _, mb := range oListTmp {
		if mb.GetNamespace() != o.Namespace {
			oList = append(oList, mb)
		}
	}

	// Read Cerebro referer
	// Add and clean finalizer to track change on Host because of there are not controller on it
	fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef=%s", o.GetName()))
	if err = r.Client.List(ctx, hostList, &client.ListOptions{Namespace: o.GetNamespace(), FieldSelector: fs}); err != nil {
		return res, errors.Wrap(err, "error when read Cerebro hosts")
	}
	for _, host := range hostList.Items {
		// Handle finalizer
		if !host.DeletionTimestamp.IsZero() {
			controllerutil.RemoveFinalizer(&host, ElasticsearchFinalizer)
			if err = r.Client.Update(ctx, &host); err != nil {
				return res, errors.Wrapf(err, "Error when add finalizer on Host %s", host.Name)
			}
			continue
		}
		if !controllerutil.ContainsFinalizer(&host, ElasticsearchFinalizer) {
			controllerutil.AddFinalizer(&host, ElasticsearchFinalizer)
			if err = r.Client.Update(ctx, &host); err != nil {
				return res, errors.Wrapf(err, "Error when add finalizer on Host %s", host.Name)
			}
		}

		cb = &cerebrocrd.Cerebro{}
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: host.Spec.CerebroRef.Namespace, Name: host.Spec.CerebroRef.Name}, cb); err != nil {
			if !k8serrors.IsNotFound(err) {
				return res, errors.Wrap(err, "Error when read cerebro")
			}
		} else {
			if cb.Namespace != o.Namespace {
				oList = append(oList, cb)
			}
		}
	}

	// Generate expected network policy
	expectedNp, err := BuildNetworkPolicy(o, oList)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate network policy")
	}
	data["expectedObject"] = expectedNp

	return res, nil
}

// Diff permit to check if network policy is up to date
func (r *NetworkPolicyReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *NetworkPolicyReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, NetworkPolicyCondition, NetworkPolicyPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *NetworkPolicyReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, NetworkPolicyCondition, NetworkPolicyPhase)
}
