package filebeat

import (
	"context"

	"github.com/pkg/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FilebeatAnnotationKey = "filebeat.k8s.webcenter.fr"
)

// GetElasticsearchRef permit to get Elasticsearch
func GetElasticsearchRef(ctx context.Context, c client.Client, fb *beatcrd.Filebeat) (es *elasticsearchcrd.Elasticsearch, err error) {
	if !fb.Spec.ElasticsearchRef.IsManaged() {
		return nil, nil
	}

	es = &elasticsearchcrd.Elasticsearch{}
	target := types.NamespacedName{Name: fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
	if fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		target.Namespace = fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace
	} else {
		target.Namespace = fb.Namespace
	}
	if err = c.Get(ctx, target, es); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read elasticsearch %s/%s", target.Namespace, target.Name)
	}

	return es, nil
}
