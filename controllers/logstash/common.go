package logstash

import (
	"context"

	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LogstashAnnotationKey = "logstash.k8s.webcenter.fr"
)

// GetElasticsearchRef permit to get Elasticsearch
func GetElasticsearchRef(ctx context.Context, c client.Client, kb *logstashcrd.Logstash) (es *elasticsearchcrd.Elasticsearch, err error) {
	if !kb.Spec.ElasticsearchRef.IsManaged() {
		return nil, nil
	}

	es = &elasticsearchcrd.Elasticsearch{}
	target := types.NamespacedName{Name: kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
	if kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		target.Namespace = kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
	} else {
		target.Namespace = kb.Namespace
	}
	if err = c.Get(ctx, target, es); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read elasticsearch %s/%s", target.Namespace, target.Name)
	}

	return es, nil
}
