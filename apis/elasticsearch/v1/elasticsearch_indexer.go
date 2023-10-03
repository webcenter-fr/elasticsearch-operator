package v1

import (
	"context"
	"fmt"

	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupIndexer setup indexer for Elasticsearch
func SetupElasticsearchIndexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.licenseSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.Spec.LicenseSecretRef != nil {
			return []string{p.Spec.LicenseSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsTlsApiEnabled() && !p.IsSelfManagedSecretForTlsApi() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			return []string{p.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.additionalVolumes.configMap.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		volumeNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.AdditionalVolumes))

		for _, volume := range p.Spec.GlobalNodeGroup.AdditionalVolumes {
			if volume.ConfigMap != nil {
				volumeNames = append(volumeNames, volume.ConfigMap.Name)
			}
		}

		return volumeNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.globalNodeGroup.additionalVolumes.secret.secretName", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		volumeNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.AdditionalVolumes))

		for _, volume := range p.Spec.GlobalNodeGroup.AdditionalVolumes {
			if volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Secret.SecretName)
			}
		}

		return volumeNames
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.env.valueFrom.configMapKeyRef.name", func(o client.Object) []string {
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
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.env.valueFrom.secretKeyRef.name", func(o client.Object) []string {
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
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.envFrom.configMapRef.name", func(o client.Object) []string {
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
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.statefulset.envFrom.secretRef.name", func(o client.Object) []string {
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
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() {
			if p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Namespace, p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name)}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.IsExternal() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		return err
	}

	return nil
}
