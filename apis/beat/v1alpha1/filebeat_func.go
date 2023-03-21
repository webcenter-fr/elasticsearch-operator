package v1alpha1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Filebeat) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsPersistence return true if persistence is enabled
func (h *Filebeat) IsPersistence() bool {
	if h.Spec.Deployment.Persistence != nil && (h.Spec.Deployment.Persistence.Volume != nil || h.Spec.Deployment.Persistence.VolumeClaimSpec != nil) {
		return true
	}

	return false
}

// IsManaged permit to know if Logstash is managed by operator
func (h FilebeatLogstashRef) IsManaged() bool {
	return h.ManagedLogstashRef != nil && h.ManagedLogstashRef.Name != ""
}

// IsExternal permit to know if Logstash is external (not managed by operator)
func (h FilebeatLogstashRef) IsExternal() bool {
	return h.ExternalLogstashRef != nil && len(h.ExternalLogstashRef.Addresses) > 0
}

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *Filebeat) IsMetricbeatMonitoring() bool {

	if h.Spec.Monitoring.Metricbeat != nil && h.Spec.Monitoring.Metricbeat.Enabled && h.Spec.Deployment.Replicas > 0 {
		return true
	}

	return false
}

// MustSetUpIndexForFilebeat setup indexer for Filebeat
func MustSetUpIndexForFilebeat(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Filebeat)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.logstashRef.managed.name", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.LogstashRef.IsManaged() {
			return []string{p.Spec.LogstashRef.ManagedLogstashRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.logstashRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.LogstashRef.IsManaged() {
			if p.Spec.LogstashRef.ManagedLogstashRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.LogstashRef.ManagedLogstashRef.Namespace, p.Spec.LogstashRef.ManagedLogstashRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.LogstashRef.ManagedLogstashRef.Namespace)}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.logstashRef.logstashCASecretRef.name", func(o client.Object) []string {
		p := o.(*Filebeat)
		if p.Spec.LogstashRef.LogstashCaSecretRef != nil {
			return []string{p.Spec.LogstashRef.LogstashCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.deployment.additionalVolumes.name", func(o client.Object) []string {
		p := o.(*Filebeat)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.deployment.env.name", func(o client.Object) []string {
		p := o.(*Filebeat)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Filebeat{}, "spec.deployment.envFrom.name", func(o client.Object) []string {
		p := o.(*Filebeat)
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
