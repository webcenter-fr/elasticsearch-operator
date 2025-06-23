package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupLogstashPipelineIndexer setup indexer for LogstashPipeline
func SetupLogstashPipelineIndexer(k8sManager manager.Manager) (err error) {
	// Index external name needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &LogstashPipeline{}, "spec.externalName", func(o client.Object) []string {
		p := o.(*LogstashPipeline)
		return []string{p.GetExternalName()}
	}); err != nil {
		return err
	}

	// Index target cluster needed by webhook to controle unicity
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &LogstashPipeline{}, "spec.targetCluster", func(o client.Object) []string {
		p := o.(*LogstashPipeline)
		return []string{p.Spec.KibanaRef.GetTargetCluster(p.Namespace)}
	}); err != nil {
		return err
	}

	return nil
}
