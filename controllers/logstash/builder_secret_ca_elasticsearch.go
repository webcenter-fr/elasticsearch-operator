package logstash

import (
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCAElasticsearchSecret permit to build CA secret from Elasticsearch ApiPKI
func BuildCAElasticsearchSecret(ls *logstashcrd.Logstash, secretCaElasticsearch *corev1.Secret) (s *corev1.Secret, err error) {

	if secretCaElasticsearch == nil {
		return nil, nil
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForCAElasticsearch(ls),
			Namespace:   ls.Namespace,
			Labels:      getLabels(ls),
			Annotations: getAnnotations(ls),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": secretCaElasticsearch.Data["ca.crt"],
		},
	}

	return s, nil
}
