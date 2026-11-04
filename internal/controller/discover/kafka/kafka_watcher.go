package kafka

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

// watchKafka permit to update if KafkaRef change
func watchKafka(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKafkas *discovercrd.KafkaList
			fs         fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// KafkaRef
		listKafkas = &discovercrd.KafkaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.kafkaRef.managed.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listKafkas, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKafkas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchKafkaUser permit to update if KafkaUser change
func watchKafkaUser(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKafkas *discovercrd.KafkaList
			fs         fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// KafkaRef
		listKafkas = &discovercrd.KafkaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.kafkaRef.userRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listKafkas, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKafkas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}

// watchSecret permit to update Kafka discover if secretRef change
func watchSecret(c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var (
			listKafkas *discovercrd.KafkaList
			fs         fields.Selector
		)

		reconcileRequests := make([]reconcile.Request, 0)

		// Kafka user secret
		listKafkas = &discovercrd.KafkaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.kafkaRef.userSecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listKafkas, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKafkas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		// Kafka ca secret
		listKafkas = &discovercrd.KafkaList{}
		fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.kafkaRef.kafkaCASecretRef.fullname=%s/%s", a.GetNamespace(), a.GetName()))
		if err := c.List(context.Background(), listKafkas, &client.ListOptions{FieldSelector: fs}); err != nil {
			panic(err)
		}
		for _, k := range listKafkas.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.Name, Namespace: k.Namespace}})
		}

		return reconcileRequests
	}
}
