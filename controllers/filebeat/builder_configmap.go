package filebeat

import (
	"strings"

	"github.com/pkg/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config maps
func BuildConfigMaps(fb *beatcrd.Filebeat, es *elasticsearchcrd.Elasticsearch) (configMaps []corev1.ConfigMap, err error) {

	configMaps = make([]corev1.ConfigMap, 0, 1)
	var cm *corev1.ConfigMap

	// ConfigMap that store configs
	var expectedConfig map[string]string

	var filebeatConf strings.Builder

	// Static config
	filebeatConf.WriteString(`
http.enabled: true
http.host: 0.0.0.0
filebeat.config.modules:
  path: ${path.config}/modules.d/*.yml
`)

	// Logstash output
	if fb.Spec.LogstashRef.IsExternal() || fb.Spec.LogstashRef.IsManaged() {
		if fb.Spec.LogstashRef.LogstashCaSecretRef == nil {
			filebeatConf.WriteString(`
output.logstash:
  hosts: '${LOGSTASH_HOST}'
  loadbalance: true
`)
		} else {
			filebeatConf.WriteString(`
output.logstash:
  hosts: '${LOGSTASH_HOST}'
  loadbalance: true
  ssl:
    enable: true
    certificate_authorities: '${LOGSTASH_CA_PATH}'
`)
		}

	}

	// Elasticsearch output
	if fb.Spec.ElasticsearchRef.IsExternal() || fb.Spec.ElasticsearchRef.IsManaged() {
		if fb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil || (es != nil && es.IsTlsApiEnabled()) {
			filebeatConf.WriteString(`
output.elasticsearch:
  hosts: '${ELASTICSEARCH_HOST}'
  ssl:
    enable: true
    certificate_authorities: '${ELASTICSEARCH_CA_PATH}'
`)

		} else {
			filebeatConf.WriteString(`
output.elasticsearch:
  hosts: '${ELASTICSEARCH_HOST}'
`)
		}
	}

	injectedConfigMap := map[string]string{
		"filebeat.yml": filebeatConf.String(),
	}

	// Inject computed config
	if fb.Spec.Config != nil {
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, fb.Spec.Config)
	} else {
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, map[string]string{"filebeat.yml": ""})
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error when merge expected config with computed config")
	}

	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   fb.Namespace,
			Name:        GetConfigMapConfigName(fb),
			Labels:      getLabels(fb),
			Annotations: getAnnotations(fb),
		},
		Data: expectedConfig,
	}

	configMaps = append(configMaps, *cm)

	// ConfigMap that store modules
	if len(fb.Spec.Module) > 0 {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetConfigMapModuleName(fb),
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Data: fb.Spec.Module,
		}

		configMaps = append(configMaps, *cm)
	}

	return configMaps, nil
}
