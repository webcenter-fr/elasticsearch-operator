package kibana

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/elastic/go-ucfg"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config map
func buildConfigMaps(kb *kibanacrd.Kibana, es *elasticsearchcrd.Elasticsearch) (configMaps []corev1.ConfigMap, err error) {
	var (
		expectedConfig map[string]string
	)

	configMaps = make([]corev1.ConfigMap, 0, 1)

	injectedConfigMap := map[string]string{}

	if kb.Spec.Tls.IsTlsEnabled() {
		injectedConfigMap["kibana.yml"] = `
server.ssl.enabled: true
server.ssl.certificate: /usr/share/kibana/config/api-cert/tls.crt
server.ssl.key: /usr/share/kibana/config/api-cert/tls.key
`
	} else {
		injectedConfigMap["kibana.yml"] = "server.ssl.enabled: false\n"
	}

	if (es != nil && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls()) || (es != nil && es.Spec.Tls.IsTlsEnabled() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (es == nil && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		injectedConfigMap["kibana.yml"] += `
elasticsearch.ssl.verificationMode: full
elasticsearch.ssl.certificateAuthorities:
  - /usr/share/kibana/config/es-ca/ca.crt
`
	}

	if kb.Spec.Endpoint.IsIngressEnabled() {
		var path string
		path, err = localhelper.GetSetting("server.basePath", []byte(kb.Spec.Config["kibana.yml"]))
		if err != nil && ucfg.ErrMissing != err {
			return nil, errors.Wrap(err, "Error when search property 'server.basePath' on kibana setting")
		}
		scheme := "https"
		if !kb.Spec.Tls.IsTlsEnabled() && kb.Spec.Endpoint.Ingress.SecretRef == nil {
			scheme = "http"
		}
		injectedConfigMap["kibana.yml"] += fmt.Sprintf("server.publicBaseUrl: %s://%s%s\n", scheme, kb.Spec.Endpoint.Ingress.Host, path)
	}

	// Inject computed config
	expectedConfig, err = helper.MergeSettings(injectedConfigMap, kb.Spec.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when merge expected config with computed config")
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetConfigMapName(kb),
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Data: expectedConfig,
	}

	configMaps = append(configMaps, *configMap)

	return configMaps, nil
}
