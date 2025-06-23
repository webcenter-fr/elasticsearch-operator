package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupComponentTemplateIndexer setup indexer for ComponentTemplate
func SetupComponentTemplateIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &ComponentTemplate{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*ComponentTemplate)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &ComponentTemplate{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*ComponentTemplate)
		return []string{p.Spec.ElasticsearchRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
