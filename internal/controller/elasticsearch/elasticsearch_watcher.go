package elasticsearch

import (
	"context"
	"fmt"

	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// watchElasticsearch permit to update if ElasticsearchRef change
func watchElasticsearchMonitoring(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listElasticsearchs *elasticsearchcrd.ElasticsearchList
			fs                 fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// ElasticsearchRef
		listElasticsearchs = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.monitoring.metricbeat.elasticsearchRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listElasticsearchs, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearchs.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchConfigMap permit to update if configMapRef change
func watchConfigMap(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listElasticsearch *elasticsearchcrd.ElasticsearchList
			fs                fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Additional volumes configMap
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.globalNodeGroup.additionalVolumes.configMap.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// Env of type configMap
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.statefulset.env.valueFrom.configMapKeyRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// EnvFrom of type configMap
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.statefulset.envFrom.configMapRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		return reconcileRequests
	}
}

// watchSecret permit to update elasticsearch if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listElasticsearch *elasticsearchcrd.ElasticsearchList
			fs                fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// License secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.licenseSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// Keystore secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.globalNodeGroup.keystoreSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// cacerts secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.globalNodeGroup.cacertsSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// TLS secret
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.tls.certificateSecretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// Additional volumes secrets
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.globalNodeGroup.additionalVolumes.secret.secretName=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// Env of type secrets
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.statefulset.env.valueFrom.secretKeyRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// EnvFrom of type secrets
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.statefulset.envFrom.secretRef.name=%s", a.GetName()))
		// Get all elasticsearch linked with secret
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, e := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: e.Name, Namespace: e.Namespace}})
		}

		// Elasticsearch API cert secret when external
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.monitoring.metricbeat.elasticsearchRef.elasticsearchCASecretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Elasticsearch credentials when external
		listElasticsearch = &elasticsearchcrd.ElasticsearchList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.monitoring.metricbeat.elasticsearchRef.secretRef.name=%s", a.GetName()))
		if err := c.List(context.Background(), listElasticsearch, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listElasticsearch.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchHost permit to update networkpolicy to allow cerebro access on Elasticsearch
func watchHost(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		o := a.(*cerebrocrd.Host)

		reconcileRequests := make([]reconcile.Request, 0)

		if o.Spec.ElasticsearchRef.IsManaged() {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: o.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name, Namespace: o.Namespace}})
		}

		return reconcileRequests
	}
}
