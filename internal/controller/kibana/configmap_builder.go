package kibana

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/elastic/go-ucfg"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildConfigMap permit to generate config map
func buildConfigMaps(kb *kibanacrd.Kibana, es *elasticsearchcrd.Elasticsearch) (configMaps []*corev1.ConfigMap, err error) {
	var expectedConfig map[string]string

	configMaps = make([]*corev1.ConfigMap, 0, 1)

	configs := map[string]string{
		"kibana.yml": "",
	}

	kibanaConf := map[string]any{}

	if kb.Spec.Tls.IsTlsEnabled() {
		kibanaConf["server.ssl.enabled"] = true
		kibanaConf["server.ssl.certificate"] = "/usr/share/kibana/config/api-cert/tls.crt"
		kibanaConf["server.ssl.key"] = "/usr/share/kibana/config/api-cert/tls.key"
	} else {
		kibanaConf["server.ssl.enabled"] = false
	}

	if (es != nil && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls()) || (es != nil && es.Spec.Tls.IsTlsEnabled() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (es == nil && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		kibanaConf["elasticsearch.ssl.verificationMode"] = "full"
		kibanaConf["elasticsearch.ssl.certificateAuthorities"] = []string{"/usr/share/kibana/config/es-ca/ca.crt"}
	}

	config, err := yaml.Marshal(kb.Spec.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when unmarshall config")
	}

	if kb.Spec.Endpoint.IsIngressEnabled() {
		var path string
		path, err = localhelper.GetSetting("server.basePath", config)
		if err != nil && ucfg.ErrMissing != err {
			return nil, errors.Wrap(err, "Error when search property 'server.basePath' on kibana setting")
		}
		scheme := "https"
		if !kb.Spec.Tls.IsTlsEnabled() && kb.Spec.Endpoint.Ingress.SecretRef == nil {
			scheme = "http"
		}
		kibanaConf["server.publicBaseUrl"] = fmt.Sprintf("%s://%s%s", scheme, kb.Spec.Endpoint.Ingress.Host, path)
	}

	injectedConfigMap := map[string]string{
		"kibana.yml": localhelper.ToYamlOrDie(kibanaConf),
	}

	if kb.Spec.Config != nil && kb.Spec.Config.Data != nil {
		config, err := yaml.Marshal(kb.Spec.Config.Data)
		if err != nil {
			return nil, errors.Wrap(err, "Error when unmarshall config")
		}
		configs["kibana.yml"] = string(config)
	}
	if kb.Spec.ExtraConfigs != nil {
		configs, err = localhelper.MergeSettings(configs, kb.Spec.ExtraConfigs)
		if err != nil {
			return nil, errors.Wrap(err, "Error when merge config and extra configs")
		}
	}

	// Inject computed config
	expectedConfig, err = localhelper.MergeSettings(configs, injectedConfigMap)
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

	configMaps = append(configMaps, configMap)

	return configMaps, nil
}
