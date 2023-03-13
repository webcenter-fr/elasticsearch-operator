package metricbeat

import (
	"context"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	MetricbeatAnnotationKey = "metricbeat.k8s.webcenter.fr"
)

func MustSetUpIndex(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.deployment.additionalVolumes.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.deployment.env.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &beatcrd.Metricbeat{}, "spec.deployment.envFrom.name", func(o client.Object) []string {
		p := o.(*beatcrd.Metricbeat)
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
