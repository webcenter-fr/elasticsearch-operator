package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupUserSpaceIndexer setup indexer for UserSpace
func SetupUserSpaceIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &UserSpace{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*UserSpace)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &UserSpace{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*UserSpace)
		return []string{p.Spec.KibanaRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
