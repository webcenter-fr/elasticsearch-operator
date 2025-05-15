package elasticsearch

import (
	"emperror.dev/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildUserSystem permit to generate system users
func buildLicenses(es *elasticsearchcrd.Elasticsearch, s *corev1.Secret) (licenses []*elasticsearchapicrd.License, err error) {
	if es.Spec.LicenseSecretRef == nil || es.Spec.LicenseSecretRef.Name == "" {
		return nil, nil
	}

	if len(s.Data["license"]) == 0 {
		return nil, errors.New("The secret must have `license` key")
	}

	licenses = []*elasticsearchapicrd.License{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetLicenseName(es),
				Labels:      getLabels(es),
				Annotations: getAnnotations(es),
			},
			Spec: elasticsearchapicrd.LicenseSpec{
				ElasticsearchRef: shared.ElasticsearchRef{
					ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
						Name: es.Name,
					},
				},
				SecretRef: &corev1.LocalObjectReference{
					Name: s.Name,
				},
			},
		},
	}

	return licenses, nil
}
