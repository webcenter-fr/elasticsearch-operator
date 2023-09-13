package v1

import (
	"context"
	"fmt"

	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Host) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}

// MustSetUpIndex setup indexer for Host
func MustSetUpIndexHost(k8sManager manager.Manager) {

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Host{}, "spec.cerebroRef.fullname", func(o client.Object) []string {
		p := o.(*Host)
		var namespace string

		if p.Spec.CerebroRef.Namespace != "" {
			namespace = p.Spec.CerebroRef.Namespace
		} else {
			namespace = p.Namespace
		}

		return []string{
			fmt.Sprintf("%s/%s", namespace, p.Spec.CerebroRef.Name),
		}

	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Host{}, "spec.elasticsearchRef", func(o client.Object) []string {
		p := o.(*Host)

		return []string{p.Spec.ElasticsearchRef}
	}); err != nil {
		panic(err)
	}
}
