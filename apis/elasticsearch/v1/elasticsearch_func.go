package v1

import (
	"context"
	"fmt"

	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// IsSelfManagedSecretForTlsApi return true if the operator manage the certificates for Api layout
// It return false if secret is provided
func (h *Elasticsearch) IsSelfManagedSecretForTlsApi() bool {
	return h.Spec.Tls.CertificateSecretRef == nil
}

// IsTlsApiEnabled return true if TLS is enabled on API endpoint
func (h *Elasticsearch) IsTlsApiEnabled() bool {
	if h.Spec.Tls.Enabled != nil && !*h.Spec.Tls.Enabled {
		return false
	}
	return true
}

// IsIngressEnabled return true if ingress is enabled
func (h *Elasticsearch) IsIngressEnabled() bool {
	if h.Spec.Endpoint.Ingress != nil && h.Spec.Endpoint.Ingress.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h *Elasticsearch) IsLoadBalancerEnabled() bool {
	if h.Spec.Endpoint.LoadBalancer != nil && h.Spec.Endpoint.LoadBalancer.Enabled {
		return true
	}

	return false
}

// IsSetVMMaxMapCount return true if SetVMMaxMapCount is enabled
func (h *Elasticsearch) IsSetVMMaxMapCount() bool {
	if h.Spec.SetVMMaxMapCount != nil && !*h.Spec.SetVMMaxMapCount {
		return false
	}

	return true
}

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Elasticsearch) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *Elasticsearch) IsMetricbeatMonitoring() bool {

	if h.Spec.Monitoring.Metricbeat != nil && h.Spec.Monitoring.Metricbeat.Enabled {
		return true
	}

	return false
}

// IsPersistence return true if persistence is enabled
func (h ElasticsearchNodeGroupSpec) IsPersistence() bool {
	if h.Persistence != nil && (h.Persistence.Volume != nil || h.Persistence.VolumeClaimSpec != nil) {
		return true
	}

	return false
}

// MustSetUpIndex setup indexer for Elasticsearch
func MustSetUpIndex(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.licenseSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.Spec.LicenseSecretRef != nil {
			return []string{p.Spec.LicenseSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsTlsApiEnabled() && !p.IsSelfManagedSecretForTlsApi() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			return []string{p.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.additionalVolumes.configMap.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		volumeNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.AdditionalVolumes))

		for _, volume := range p.Spec.GlobalNodeGroup.AdditionalVolumes {
			if volume.ConfigMap != nil {
				volumeNames = append(volumeNames, volume.ConfigMap.Name)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.additionalVolumes.secret.secretName", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		volumeNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.AdditionalVolumes))

		for _, volume := range p.Spec.GlobalNodeGroup.AdditionalVolumes {
			if volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Secret.SecretName)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.env.valueFrom.configMapKeyRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		envNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.Env))

		for _, env := range p.Spec.GlobalNodeGroup.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, env := range nodeGroup.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					envNames = append(envNames, env.ValueFrom.ConfigMapKeyRef.Name)
				}
			}
		}

		return funk.UniqString(envNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.env.valueFrom.secretKeyRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		envNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.Env))

		for _, env := range p.Spec.GlobalNodeGroup.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				envNames = append(envNames, env.ValueFrom.SecretKeyRef.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, env := range nodeGroup.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					envNames = append(envNames, env.ValueFrom.SecretKeyRef.Name)
				}
			}
		}

		return funk.UniqString(envNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.envFrom.configMapRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		envFromNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.EnvFrom))

		for _, envFrom := range p.Spec.GlobalNodeGroup.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, envFrom := range nodeGroup.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
				}
			}
		}

		return funk.UniqString(envFromNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.envFrom.secretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		envFromNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.EnvFrom))

		for _, envFrom := range p.Spec.GlobalNodeGroup.EnvFrom {
			if envFrom.SecretRef != nil {
				envFromNames = append(envFromNames, envFrom.SecretRef.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, envFrom := range nodeGroup.EnvFrom {
				if envFrom.SecretRef != nil {
					envFromNames = append(envFromNames, envFrom.SecretRef.Name)
				}
			}
		}

		return funk.UniqString(envFromNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() {
			if p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Namespace, p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name)}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.IsExternal() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}
}
