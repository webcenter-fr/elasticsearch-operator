package discover

import (
	"context"
	"fmt"
	"reflect"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/sentinel"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// WatchDiscoverSecret permit to watch discover secrets and reconcile workloads that reference them
func WatchDiscoverSecret[k8sObjectList client.ObjectList, k8sObject client.Object](c client.Client) handler.MapFunc {
	return func(ctx context.Context, a client.Object) []reconcile.Request {
		var fs fields.Selector
		var tmpList k8sObjectList
		listObjects := reflect.New(reflect.TypeOf(tmpList).Elem()).Interface().(k8sObjectList)
		reconcileRequests := make([]reconcile.Request, 0)

		if a.GetLabels() != nil && a.GetLabels()[discovercrd.DiscoverAnnotationKey] == "true" && a.GetLabels()["discoverName"] != "" {
			fs = fields.ParseSelectorOrDie(fmt.Sprintf("spec.discover.name=%s", a.GetLabels()["discoverName"]))
			if err := c.List(context.Background(), listObjects, &client.ListOptions{FieldSelector: fs, Namespace: a.GetNamespace()}); err != nil {
				panic(err)
			}
			for _, k := range sentinel.GetItems[k8sObjectList, k8sObject](listObjects) {
				reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: k.GetName(), Namespace: k.GetNamespace()}})
			}
		}

		return reconcileRequests
	}
}
