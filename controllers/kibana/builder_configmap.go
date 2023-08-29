package kibana

import (
	"fmt"

	"emperror.dev/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config map
func BuildConfigMap(kb *kibanacrd.Kibana, es *elasticsearchcrd.Elasticsearch) (configMap *corev1.ConfigMap, err error) {
	var (
		expectedConfig map[string]string
	)

	injectedConfigMap := map[string]string{}

	if kb.IsTlsEnabled() {
		injectedConfigMap["kibana.yml"] = `
server.ssl.enabled: true
server.ssl.certificate: /usr/share/kibana/config/api-cert/tls.crt
server.ssl.key: /usr/share/kibana/config/api-cert/tls.key
`
	} else {
		injectedConfigMap["kibana.yml"] = "server.ssl.enabled: false\n"
	}

	if (es != nil && es.IsTlsApiEnabled() && es.IsSelfManagedSecretForTlsApi()) || (es != nil && es.IsTlsApiEnabled() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (es == nil && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		injectedConfigMap["kibana.yml"] += `
elasticsearch.ssl.verificationMode: full
elasticsearch.ssl.certificateAuthorities:
  - /usr/share/kibana/config/es-ca/ca.crt
`
	}

	if kb.IsIngressEnabled() {
		scheme := "https"
		if !kb.IsTlsEnabled() && kb.Spec.Endpoint.Ingress.SecretRef == nil {
			scheme = "http"
		}
		injectedConfigMap["kibana.yml"] += fmt.Sprintf("server.publicBaseUrl: %s://%s\n", scheme, kb.Spec.Endpoint.Ingress.Host)
	}

	// Inject computed config
	expectedConfig, err = helper.MergeSettings(injectedConfigMap, kb.Spec.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when merge expected config with computed config")
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetConfigMapName(kb),
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Data: expectedConfig,
	}

	return configMap, nil
}
