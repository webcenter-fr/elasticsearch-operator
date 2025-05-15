package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCALogstashSecret permit to build CA secret from Logstash
func buildCALogstashSecrets(fb *beatcrd.Filebeat, secretCaLogstash *corev1.Secret) (secrets []*corev1.Secret, err error) {
	if secretCaLogstash == nil {
		return nil, nil
	}

	secrets = []*corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetSecretNameForCALogstash(fb),
				Namespace:   fb.Namespace,
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"ca.crt": secretCaLogstash.Data["ca.crt"],
			},
		},
	}

	return secrets, nil
}
