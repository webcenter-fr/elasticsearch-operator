package logstash

import (
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCredentialSecret permit to build credential secret from Elasticsearch credentials
func BuildCredentialSecret(ls *logstashcrd.Logstash, secretCredentials *corev1.Secret) (s *corev1.Secret, err error) {

	if secretCredentials == nil {
		return nil, nil
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForCredentials(ls),
			Namespace:   ls.Namespace,
			Labels:      getLabels(ls),
			Annotations: getAnnotations(ls),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"logstash_system":        secretCredentials.Data["logstash_system"],
			"remote_monitoring_user": secretCredentials.Data["remote_monitoring_user"],
		},
	}

	return s, nil
}
