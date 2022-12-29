package kibana

import (
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCredentialSecret permit to build credential secret from Elasticsearch credentials
func BuildCredentialSecret(kb *kibanaapi.Kibana, secretCredentials *corev1.Secret) (s *corev1.Secret, err error) {

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForCredentials(kb),
			Namespace:   kb.Namespace,
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kibana_system": secretCredentials.Data["kibana_system"],
		},
	}

	return s, nil
}
