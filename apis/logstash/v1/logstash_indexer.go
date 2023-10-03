package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// MustSetUpIndex setup indexer for Logstash
func SetupLogstashIndexer(k8sManager manager.Manager) (err error) {
	// Add indexers on Logstash to track secret change
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.ElasticsearchRef.IsManaged() {
			if p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace, p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.additionalVolumes.configMap.name", func(o client.Object) []string {
		p := o.(*Logstash)
		volumeNames := make([]string, 0, len(p.Spec.Deployment.AdditionalVolumes))

		for _, volume := range p.Spec.Deployment.AdditionalVolumes {
			if volume.ConfigMap != nil {
				volumeNames = append(volumeNames, volume.ConfigMap.Name)
			}
		}

		return volumeNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.additionalVolumes.secret.secretName", func(o client.Object) []string {
		p := o.(*Logstash)
		volumeNames := make([]string, 0, len(p.Spec.Deployment.AdditionalVolumes))

		for _, volume := range p.Spec.Deployment.AdditionalVolumes {
			if volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Secret.SecretName)
			}
		}

		return volumeNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.env.valueFrom.configMapKeyRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		envNames := make([]string, 0, len(p.Spec.Deployment.Env))

		for _, env := range p.Spec.Deployment.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}

		return envNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.env.valueFrom.secretKeyRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		envNames := make([]string, 0, len(p.Spec.Deployment.Env))

		for _, env := range p.Spec.Deployment.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.SecretKeyRef.Name)
			}
		}

		return envNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.envFrom.configMapRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		envFromNames := make([]string, 0, len(p.Spec.Deployment.EnvFrom))

		for _, envFrom := range p.Spec.Deployment.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
			}
		}

		return envFromNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.deployment.envFrom.secretRef.name", func(o client.Object) []string {
		p := o.(*Logstash)
		envFromNames := make([]string, 0, len(p.Spec.Deployment.EnvFrom))

		for _, envFrom := range p.Spec.Deployment.EnvFrom {
			if envFrom.SecretRef != nil {
				envFromNames = append(envFromNames, envFrom.SecretRef.Name)
			}
		}

		return envFromNames
	}); err != nil {
		return err
	}

	return nil
}
