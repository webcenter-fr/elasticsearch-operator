package kibana

import (
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCredentialSecret permit to build credential secret from Elasticsearch credentials
func buildCredentialSecrets(kb *kibanacrd.Kibana, secretCredentials *corev1.Secret) (secrets []corev1.Secret, err error) {
	if secretCredentials == nil {
		return nil, nil
	}

	secrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetSecretNameForCredentials(kb),
				Namespace:   kb.Namespace,
				Labels:      getLabels(kb),
				Annotations: getAnnotations(kb),
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"kibana_system":          secretCredentials.Data["kibana_system"],
				"remote_monitoring_user": secretCredentials.Data["remote_monitoring_user"],
				"username":               []byte("kibana_system"),
			},
		},
	}

	return secrets, nil
}
