package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupRoleMappingIndexer setup indexer for RoleMapping
func SetupRoleMappingIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &RoleMapping{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*RoleMapping)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &RoleMapping{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*RoleMapping)
		return []string{p.Spec.ElasticsearchRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
