package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupUserIndexexer setup indexer for User
func SetupUserIndexexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &User{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*User)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	return nil
}
