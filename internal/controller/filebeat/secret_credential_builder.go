package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCredentialSecret permit to build credential secret from Elasticsearch credentials
func buildCredentialSecrets(fb *beatcrd.Filebeat, secretCredentials *corev1.Secret) (secrets []corev1.Secret, err error) {
	if secretCredentials == nil {
		return nil, nil
	}

	secrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetSecretNameForCredentials(fb),
				Namespace:   fb.Namespace,
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"beats_system":           secretCredentials.Data["beats_system"],
				"remote_monitoring_user": secretCredentials.Data["remote_monitoring_user"],
			},
		},
	}

	return secrets, nil
}
