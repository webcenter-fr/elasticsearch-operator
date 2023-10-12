package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupLicenceIndexer setuyp the indexer for licence
func SetupLicenceIndexer(k8sManager manager.Manager) (err error) {
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
