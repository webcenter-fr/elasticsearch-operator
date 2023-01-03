package elasticsearch

import (
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// BuildUserSystem permit to generate system users
func BuildUserSystem(es *elasticsearchcrd.Elasticsearch, s *corev1.Secret) (users []elasticsearchapicrd.User, err error) {

	users = make([]elasticsearchapicrd.User, 0, len(s.Data))

	for key, _ := range s.Data {
		if key != "elastic" {
			user := elasticsearchapicrd.User{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   es.Namespace,
					Name:        GetUserSystemName(es, key),
					Labels:      getLabels(es),
					Annotations: getAnnotations(es),
				},
				Spec: elasticsearchapicrd.UserSpec{
					ElasticsearchRefSpec: elasticsearchapicrd.ElasticsearchRefSpec{
						Name: es.Name,
					},
					Enabled:  true,
					Username: key,
					SecretRef: &elasticsearchapicrd.UserSecret{
						Name: s.Name,
						Key:  key,
					},
					IsProtected: pointer.Bool(true),
				},
			}

			users = append(users, user)
		}
	}

	return users, nil
}
