package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupLicenceIndexer setuyp the indexer for licence
func SetupLicenceIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &License{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*License)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &License{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*License)
		return []string{p.Spec.ElasticsearchRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	// Index secret ref
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &License{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*License)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	return nil
}
