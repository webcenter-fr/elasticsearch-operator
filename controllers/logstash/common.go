package logstash

import (
	"context"

	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	LogstashAnnotationKey = "logstash.k8s.webcenter.fr"
)

func MustSetUpIndex(k8sManager manager.Manager) {
	// Add indexers on Logstash to track secret change
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.deployment.additionalVolumes.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.deployment.env.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
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

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &logstashcrd.Logstash{}, "spec.deployment.envFrom.name", func(o client.Object) []string {
		p := o.(*logstashcrd.Logstash)
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

// GetElasticsearchRef permit to get Elasticsearch
func GetElasticsearchRef(ctx context.Context, c client.Client, ls *logstashcrd.Logstash) (es *elasticsearchcrd.Elasticsearch, err error) {
	if !ls.Spec.ElasticsearchRef.IsManaged() {
		return nil, nil
	}

	es = &elasticsearchcrd.Elasticsearch{}
	target := types.NamespacedName{Name: ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
	if ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		target.Namespace = ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
	} else {
		target.Namespace = ls.Namespace
	}
	if err = c.Get(ctx, target, es); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read elasticsearch %s/%s", target.Namespace, target.Name)
	}

	return es, nil
}
