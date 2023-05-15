package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// IsSelfManagedSecretForTls return true if the operator manage the certificates for TLS
// It return false if secret is provided
func (h *Kibana) IsSelfManagedSecretForTls() bool {
	return h.Spec.Tls.CertificateSecretRef == nil
}

// IsTlsEnabled return true if TLS is enabled to access on Kibana
func (h *Kibana) IsTlsEnabled() bool {
	if h.Spec.Tls.Enabled != nil && !*h.Spec.Tls.Enabled {
		return false
	}
	return true
}

// IsIngressEnabled return true if ingress is enabled
func (h *Kibana) IsIngressEnabled() bool {
	if h.Spec.Endpoint.Ingress != nil && h.Spec.Endpoint.Ingress.Enabled {
		return true
	}

	return false
}

// IsLoadBalancerEnabled return true if LoadBalancer is enabled
func (h *Kibana) IsLoadBalancerEnabled() bool {
	if h.Spec.Endpoint.LoadBalancer != nil && h.Spec.Endpoint.LoadBalancer.Enabled {
		return true
	}

	return false
}

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h *Kibana) IsPrometheusMonitoring() bool {

	if h.Spec.Monitoring.Prometheus != nil && h.Spec.Monitoring.Prometheus.Enabled {
		return true
	}

	return false
}

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *Kibana) IsMetricbeatMonitoring() bool {

	if h.Spec.Monitoring.Metricbeat != nil && h.Spec.Monitoring.Metricbeat.Enabled && h.Spec.Deployment.Replicas > 0 {
		return true
	}

	return false
}

// IsPdb return true if PDB is enabled
func (h *Kibana) IsPdb() bool {
	if h.Spec.Deployment.PodDisruptionBudgetSpec != nil || h.Spec.Deployment.Replicas > 1 {
		return true
	}

	return false
}

// MustSetUpIndex setup indexer for kibana
func MustSetUpIndex(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
		if p.IsTlsEnabled() && !p.IsSelfManagedSecretForTls() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Kibana)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Kibana)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.deployment.env.valueFrom.configMapKeyRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.deployment.env.valueFrom.secretKeyRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.deployment.envFrom.configMapRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kibana{}, "spec.deployment.envFrom.secretRef.name", func(o client.Object) []string {
		p := o.(*Kibana)
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
