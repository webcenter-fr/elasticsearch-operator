package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupElasticsearchIndexer setup indexer for Elasticsearch
func SetupElasticsearchIndexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.elasticsearchRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Elasticsearch)
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

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.elasticsearchRef.secretRef.name", func(o client.Object) []string {
		p := o.(*Elasticsearch)

		if p.Spec.ElasticsearchRef.SecretRef != nil {
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.ElasticsearchRef.SecretRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.elasticsearchRef.userSecretRef.fullname", func(o client.Object) []string {
		p := o.(*Elasticsearch)

		if p.Spec.ElasticsearchRef.IsManaged() && p.Status.ElasticsearchUserSecretRef != nil {
			ns := p.Namespace
			if p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
				ns = p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", ns, *p.Status.ElasticsearchUserSecretRef)}
		}

		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.SecretRef != nil {
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.ElasticsearchRef.SecretRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Elasticsearch{}, "spec.elasticsearchRef.elasticsearchCASecretRef.fullname", func(o client.Object) []string {
		p := o.(*Elasticsearch)
		res := make([]string, 0, 1)

		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			res = append(res, fmt.Sprintf("%s/%s", p.Namespace, p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name))
		}

		if p.Spec.ElasticsearchRef.IsManaged() && p.Status.ElasticsearchCASecretRef != nil {
			ns := p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
			if p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace == "" {
				ns = p.Namespace
			}
			res = append(res, fmt.Sprintf("%s/%s", ns, *p.Status.ElasticsearchCASecretRef))
		}

		return res
	}); err != nil {
		return err
	}

	return nil
}
