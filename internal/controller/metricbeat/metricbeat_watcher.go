package metricbeat

import (
	"context"
	"fmt"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// watchElasticsearch permit to update if ElasticsearchRef change
func watchElasticsearch(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchConfigMap permit to update if configMapRef change
func watchConfigMap(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Additional volumes secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.additionalVolumes.configMap.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Env of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.configMapKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.configMapRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchSecret permit to update Metricbeat if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listMetricbeats *beatcrd.MetricbeatList
			fs              fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Elasticsearch API cert secret when external
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.elasticsearchCASecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch credentials when external
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Additional volumes secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.additionalVolumes.secret.secretName=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Env of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.env.valueFrom.secretKeyRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// EnvFrom of type secrets
		listMetricbeats = &beatcrd.MetricbeatList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.deployment.envFrom.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listMetricbeats, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listMetricbeats.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}
