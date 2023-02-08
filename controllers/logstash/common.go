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
