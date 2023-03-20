package v1alpha1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// IsPersistence return true if persistence is enabled
func (h *Metricbeat) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaimSpec != nil) {
		return true
	}

	return false
}

// MustSetUpIndexForMetricbeat setup indexer for Metricbeat
func MustSetUpIndexForMetricbeat(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Metricbeat)
		if p.Spec.ElasticsearchRef.IsManaged() {
			if p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace, p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.additionalVolumes.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		volumeNames := make([]string, 0, len(p.Spec.Deployment.AdditionalVolumes))

		for _, volume := range p.Spec.Deployment.AdditionalVolumes {
			if volume.ConfigMap != nil || volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Name)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.env.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envNames := make([]string, 0, len(p.Spec.Deployment.Env))

		for _, env := range p.Spec.Deployment.Env {
			if env.ValueFrom != nil && (env.ValueFrom.SecretKeyRef != nil || env.ValueFrom.ConfigMapKeyRef != nil) {
				envNames = append(envNames, env.Name)
			}
		}

		return envNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.envFrom.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envFromNames := make([]string, 0, len(p.Spec.Deployment.EnvFrom))

		for _, envFrom := range p.Spec.Deployment.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
			} else if envFrom.SecretRef != nil {
				envFromNames = append(envFromNames, envFrom.SecretRef.Name)
			}
		}

		return envFromNames
	}); err != nil {
		panic(err)
	}
}
