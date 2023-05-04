package elasticsearch

import (
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCredentialSecret permit to build credential secret
func BuildCredentialSecret(o *elasticsearchcrd.Elasticsearch) (s *corev1.Secret, err error) {

	var (
		esPassword  string
		kbPassword  string
		lsPassword  string
		btPassword  string
		apmPassword string
		rmPassword  string
	)

	esPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate elastic password")
	}
	kbPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate kibana_system password")
	}
	lsPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate logstash_system password")
	}
	btPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate beats_system password")
	}
	apmPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate apm_system password")
	}
	rmPassword, err = password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate remote_monitoring_user password")
	}

	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetSecretNameForCredentials(o),
			Namespace:   o.Namespace,
			Labels:      getLabels(o),
			Annotations: getAnnotations(o),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"elastic":                []byte(esPassword),
			"kibana_system":          []byte(kbPassword),
			"logstash_system":        []byte(lsPassword),
			"beats_system":           []byte(btPassword),
			"apm_system":             []byte(apmPassword),
			"remote_monitoring_user": []byte(rmPassword),
		},
	}

	return s, nil
}
