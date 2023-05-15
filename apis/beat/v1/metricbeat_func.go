package v1

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

// IsPdb return true if PDB is enabled
func (h *Metricbeat) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.additionalVolumes.configMap.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		volumeNames := make([]string, 0, len(p.Spec.Deployment.AdditionalVolumes))

		for _, volume := range p.Spec.Deployment.AdditionalVolumes {
			if volume.ConfigMap != nil {
				volumeNames = append(volumeNames, volume.ConfigMap.Name)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.additionalVolumes.secret.secretName", func(o client.Object) []string {
		p := o.(*Metricbeat)
		volumeNames := make([]string, 0, len(p.Spec.Deployment.AdditionalVolumes))

		for _, volume := range p.Spec.Deployment.AdditionalVolumes {
			if volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Secret.SecretName)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.env.valueFrom.configMapKeyRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envNames := make([]string, 0, len(p.Spec.Deployment.Env))

		for _, env := range p.Spec.Deployment.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}

		return envNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.env.valueFrom.secretKeyRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envNames := make([]string, 0, len(p.Spec.Deployment.Env))

		for _, env := range p.Spec.Deployment.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.SecretKeyRef.Name)
			}
		}

		return envNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.envFrom.configMapRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envFromNames := make([]string, 0, len(p.Spec.Deployment.EnvFrom))

		for _, envFrom := range p.Spec.Deployment.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
			}
		}

		return envFromNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Metricbeat{}, "spec.deployment.envFrom.secretRef.name", func(o client.Object) []string {
		p := o.(*Metricbeat)
		envFromNames := make([]string, 0, len(p.Spec.Deployment.EnvFrom))

		for _, envFrom := range p.Spec.Deployment.EnvFrom {
			if envFrom.SecretRef != nil {
				envFromNames = append(envFromNames, envFrom.SecretRef.Name)
			}
		}

		return envFromNames
	}); err != nil {
		panic(err)
	}
}
