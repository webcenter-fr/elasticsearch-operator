package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupHostIndexersetup indexer for Host
func SetupHostIndexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Host{}, "spec.cerebroRef.fullname", func(o client.Object) []string {
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
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Host{}, "spec.elasticsearchRef", func(o client.Object) []string {
		p := o.(*Host)

		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}

		return []string{}
	}); err != nil {
		return err
	}

	return nil
}
