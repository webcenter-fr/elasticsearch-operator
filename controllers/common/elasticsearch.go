package common

import (
	"context"

	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetElasticsearchRef permit to get Elasticsearch
func GetElasticsearchFromRef(ctx context.Context, c client.Client, o client.Object, esRef shared.ElasticsearchRef) (es *elasticsearchcrd.Elasticsearch, err error) {
	if !esRef.IsManaged() {
		return nil, nil
	}

	es = &elasticsearchcrd.Elasticsearch{}
	target := types.NamespacedName{Name: esRef.ManagedElasticsearchRef.Name}
	if esRef.ManagedElasticsearchRef.Namespace != "" {
		target.Namespace = esRef.ManagedElasticsearchRef.Namespace
	} else {
		target.Namespace = o.GetNamespace()
	}
	if err = c.Get(ctx, target, es); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read elasticsearch %s/%s", target.Namespace, target.Name)
	}

	return es, nil
}
