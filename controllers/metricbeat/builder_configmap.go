package metricbeat

import (
	"strings"

	"emperror.dev/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config maps
func BuildConfigMaps(mb *beatcrd.Metricbeat, es *elasticsearchcrd.Elasticsearch) (configMaps []corev1.ConfigMap, err error) {

	configMaps = make([]corev1.ConfigMap, 0, 1)
	var cm *corev1.ConfigMap

	// ConfigMap that store configs
	var expectedConfig map[string]string

	var metricbeatConf strings.Builder

	// Static config
	metricbeatConf.WriteString(`
http.enabled: true
http.host: 0.0.0.0
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
`)

	// Elasticsearch output
	if (mb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled()) || (mb.Spec.ElasticsearchRef.IsExternal() && mb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		metricbeatConf.WriteString(`
output.elasticsearch:
  hosts: '${ELASTICSEARCH_HOST}'
  username: '${METRICBEAT_USERNAME}'
  password: '${METRICBEAT_PASSWORD}'
  ssl:
    enable: true
    certificate_authorities: '/usr/share/metricbeat/es-ca/ca.crt'

`)
	} else {
		metricbeatConf.WriteString(`
output.elasticsearch:
  hosts: '${ELASTICSEARCH_HOST}'
  username: '${METRICBEAT_USERNAME}'
  password: '${METRICBEAT_PASSWORD}'
`)
	}

	injectedConfigMap := map[string]string{
		"metricbeat.yml": metricbeatConf.String(),
	}

	// Inject computed config
	if mb.Spec.Config != nil {
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, mb.Spec.Config)
	} else {
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, map[string]string{"metricbeat.yml": ""})
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error when merge expected config with computed config")
	}

	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mb.Namespace,
			Name:        GetConfigMapConfigName(mb),
			Labels:      getLabels(mb),
			Annotations: getAnnotations(mb),
		},
		Data: expectedConfig,
	}

	configMaps = append(configMaps, *cm)

	// ConfigMap that store modules
	if len(mb.Spec.Module) > 0 {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   mb.Namespace,
				Name:        GetConfigMapModuleName(mb),
				Labels:      getLabels(mb),
				Annotations: getAnnotations(mb),
			},
			Data: mb.Spec.Module,
		}

		configMaps = append(configMaps, *cm)
	}

	return configMaps, nil
}
