package logstash

import (
	"emperror.dev/errors"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config maps
func buildConfigMaps(ls *logstashcrd.Logstash) (configMaps []corev1.ConfigMap, err error) {
	configMaps = make([]corev1.ConfigMap, 0)
	var cm *corev1.ConfigMap

	// ConfigMap that store configs
	if len(ls.Spec.Config) > 0 {
		var expectedConfig map[string]string

		injectedConfigMap := map[string]string{
			"logstash.yml": "",
		}

		// Inject computed config
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, ls.Spec.Config)
		if err != nil {
			return nil, errors.Wrap(err, "Error when merge expected config with computed config")
		}

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetConfigMapConfigName(ls),
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			Data: expectedConfig,
		}

		configMaps = append(configMaps, *cm)
	}

	// ConfigMap that store pipelines
	if len(ls.Spec.Pipeline) > 0 {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetConfigMapPipelineName(ls),
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			Data: ls.Spec.Pipeline,
		}

		configMaps = append(configMaps, *cm)
	}

	// ConfigMap that store pattern
	if len(ls.Spec.Pattern) > 0 {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetConfigMapPatternName(ls),
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			Data: ls.Spec.Pattern,
		}

		configMaps = append(configMaps, *cm)
	}

	// COnfigMap for Prometheus exporter
	if ls.Spec.Monitoring.IsPrometheusMonitoring() {
		config := map[string]any{
			"logstash": map[string]any{
				"servers": []map[string]any{
					{
						"url": "http://127.0.0.1:9600",
					},
				},
				"timeout": "30s",
			},
			"server": map[string]any{
				"host": "0.0.0.0",
				"port": 9198,
			},
			"logging": map[string]any{
				"level": "info",
			},
		}

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetConfigMapExporterName(ls),
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			Data: map[string]string{
				"config.yml": helper.ToYamlOrDie(config),
			},
		}

		configMaps = append(configMaps, *cm)
	}

	return configMaps, nil
}
