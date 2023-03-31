package cerebro

import (
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildApplicationSecret permit to build credential secret
func BuildApplicationSecret(o *cerebrocrd.Cerebro) (s *corev1.Secret, err error) {

	var (
		applicationSecret string
	)

	applicationSecret, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate secret for application")
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForApplication(o),
			Namespace:   o.Namespace,
			Labels:      getLabels(o),
			Annotations: getAnnotations(o),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"application": []byte(applicationSecret),
		},
	}

	return s, nil
}