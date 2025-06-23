package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupSnapshotRepositoryIndexer setup indexer for SnapshotRepository
func SetupSnapshotRepositoryIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &SnapshotRepository{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*SnapshotRepository)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &SnapshotRepository{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*SnapshotRepository)
		return []string{p.Spec.ElasticsearchRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
