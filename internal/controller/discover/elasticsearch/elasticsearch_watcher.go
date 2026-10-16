package elasticsearch

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

// watchElasticsearch permit to update if ElasticsearchRef change
func watchElasticsearch(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listElasticsearchs *discovercrd.ElasticsearchList
			fs                 fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listElasticsearchs = &discovercrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listElasticsearchs, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearchs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchSecret permit to update Elasticsearch discover if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listElasticsearchs *discovercrd.ElasticsearchList
			fs                 fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Elasticsearch ca secret
		listElasticsearchs = &discovercrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.elasticsearchCASecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listElasticsearchs, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearchs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch user secret
		listElasticsearchs = &discovercrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.userSecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listElasticsearchs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearchs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch secretRef (credentials)
		listElasticsearchs = &discovercrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.elasticsearchRef.secretRef.name=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listElasticsearchs, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearchs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}
