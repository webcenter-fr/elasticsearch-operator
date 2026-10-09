package logstash

import (
	"context"
	"fmt"

	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// watchLogstash permit to update if LogstashRef change
func watchLogstash(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listLogstashs *discovercrd.LogstashList
			fs            fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// LogstashRef
		listLogstashs = &discovercrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.logstashRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchSecret permit to update Logstash discover if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listLogstashs *discovercrd.LogstashList
			fs            fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Logstash ca secret
		listLogstashs = &discovercrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.logstashRef.logstashCASecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Logstash user secret
		listLogstashs = &discovercrd.LogstashList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.logstashRef.userSecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listLogstashs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listLogstashs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}
