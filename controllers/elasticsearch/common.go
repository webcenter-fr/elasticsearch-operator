package elasticsearch

import (
	"context"

	"github.com/thoas/go-funk"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	ElasticsearchAnnotationKey = "elasticsearch.k8s.webcenter.fr"
)

func MustSetUpIndex(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.licenseSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.Spec.LicenseSecretRef != nil {
			return []string{p.Spec.LicenseSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsTlsApiEnabled() && !p.IsSelfManagedSecretForTlsApi() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.globalNodeGroup.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			return []string{p.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.globalNodeGroup.additionalVolumes.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		volumeNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.AdditionalVolumes))

		for _, volume := range p.Spec.GlobalNodeGroup.AdditionalVolumes {
			if volume.ConfigMap != nil || volume.Secret != nil {
				volumeNames = append(volumeNames, volume.Name)
			}
		}

		return volumeNames
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.statefulset.env.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		envNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.Env))

		for _, env := range p.Spec.GlobalNodeGroup.Env {
			if env.ValueFrom != nil && (env.ValueFrom.SecretKeyRef != nil || env.ValueFrom.ConfigMapKeyRef != nil) {
				envNames = append(envNames, env.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, env := range nodeGroup.Env {
				if env.ValueFrom != nil && (env.ValueFrom.SecretKeyRef != nil || env.ValueFrom.ConfigMapKeyRef != nil) {
					envNames = append(envNames, env.Name)
				}
			}
		}

		return funk.UniqString(envNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.statefulset.envFrom.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		envFromNames := make([]string, 0, len(p.Spec.GlobalNodeGroup.EnvFrom))

		for _, envFrom := range p.Spec.GlobalNodeGroup.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
			} else if envFrom.SecretRef != nil {
				envFromNames = append(envFromNames, envFrom.SecretRef.Name)
			}
		}

		for _, nodeGroup := range p.Spec.NodeGroups {
			for _, envFrom := range nodeGroup.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					envFromNames = append(envFromNames, envFrom.ConfigMapRef.Name)
				} else if envFrom.SecretRef != nil {
					envFromNames = append(envFromNames, envFrom.SecretRef.Name)
				}
			}
		}

		return funk.UniqString(envFromNames)
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsMetricbeatMonitoring() {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.monitoring.metricbeat.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsMetricbeatMonitoring() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.IsExternal() && p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.Monitoring.Metricbeat.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}
}
