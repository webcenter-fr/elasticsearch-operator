package elasticsearch

import (
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// BuildUserSystem permit to generate system users
func buildSystemUsers(es *elasticsearchcrd.Elasticsearch, s *corev1.Secret) (users []elasticsearchapicrd.User, err error) {

	users = make([]elasticsearchapicrd.User, 0, len(s.Data))

	for key := range s.Data {
		if key != "elastic" {
			user := elasticsearchapicrd.User{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   es.Namespace,
					Name:        GetUserSystemName(es, key),
					Labels:      getLabels(es),
					Annotations: getAnnotations(es),
				},
				Spec: elasticsearchapicrd.UserSpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: es.Name,
						},
					},
					Enabled:  true,
					Username: key,
					SecretRef: &corev1.SecretKeySelector{
						Key: key,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: s.Name,
						},
					},
					IsProtected: ptr.To[bool](true),
				},
			}

			users = append(users, user)
		}
	}

	return users, nil
}
