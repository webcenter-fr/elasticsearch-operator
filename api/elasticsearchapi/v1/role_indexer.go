package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupRoleIndexer setup indexer for Role
func SetupRoleIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Role{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*Role)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Role{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*Role)
		return []string{p.Spec.ElasticsearchRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
