package filebeat

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/logstash"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config maps
func buildConfigMaps(fb *beatcrd.Filebeat, es *elasticsearchcrd.Elasticsearch, ls *logstashcrd.Logstash, elasticsearchCASecret *corev1.Secret, logstashCASecret *corev1.Secret) (configMaps []corev1.ConfigMap, err error) {
	configMaps = make([]corev1.ConfigMap, 0, 1)
	var cm *corev1.ConfigMap

	// ConfigMap that store configs
	var expectedConfig map[string]string

	// Compute the logstash hosts
	logstashHosts := make([]string, 0, 1)

	if ls != nil {
		if fb.Spec.LogstashRef.ManagedLogstashRef.TargetService != "" {
			logstashHosts = append(logstashHosts, fmt.Sprintf("%s.%s.svc:%d", logstashcontrollers.GetServiceName(ls, fb.Spec.LogstashRef.ManagedLogstashRef.TargetService), ls.Namespace, fb.Spec.LogstashRef.ManagedLogstashRef.Port))
		} else {
			for i := 0; i < int(ls.Spec.Deployment.Replicas); i++ {
				logstashHosts = append(logstashHosts, fmt.Sprintf("%s-%d.%s.%s.svc:%d", logstashcontrollers.GetStatefulsetName(ls), i, logstashcontrollers.GetGlobalServiceName(ls), ls.Namespace, fb.Spec.LogstashRef.ManagedLogstashRef.Port))
			}
		}
	} else if fb.Spec.LogstashRef.IsExternal() && len(fb.Spec.LogstashRef.ExternalLogstashRef.Addresses) > 0 {
		logstashHosts = fb.Spec.LogstashRef.ExternalLogstashRef.Addresses
	}

	// Compute the Elasticsearch hosts
	elasticsearchHosts := make([]string, 0, 1)
	if es != nil {
		elasticsearchHosts = append(elasticsearchHosts, elasticsearchcontrollers.GetPublicUrl(es, fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup, false))
	} else if fb.Spec.ElasticsearchRef.IsExternal() && len(fb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses) > 0 {
		elasticsearchHosts = fb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses
	}

	// Static config
	filebeatConf := map[string]any{
		"http.enabled": true,
		"http.host":    "0.0.0.0",
		"filebeat.config.modules": map[string]any{
			"path": "${path.config}/modules.d/*.yml",
		},
	}

	// Logstash output
	if fb.Spec.LogstashRef.IsExternal() || fb.Spec.LogstashRef.IsManaged() {
		certificates := make([]string, 0)

		// Compute logstash pki ca
		if ls != nil && ls.Spec.Pki.IsEnabled() && ls.Spec.Pki.HasBeatCertificate() {
			certificates = append(certificates, "/usr/share/filebeat/ls-ca/ca.crt")
		}

		// Compute external certificates
		if logstashCASecret != nil {
			for certificateName := range logstashCASecret.Data {
				if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
					certificates = append(certificates, fmt.Sprintf("/usr/share/filebeat/ls-custom-ca/%s", certificateName))
				}
			}
		}
		if len(certificates) == 0 {
			filebeatConf["output.logstash"] = map[string]any{
				"hosts":       logstashHosts,
				"loadbalance": true,
			}
		} else {
			filebeatConf["output.logstash"] = map[string]any{
				"hosts":       logstashHosts,
				"loadbalance": true,
				"ssl": map[string]any{
					"enable":                  true,
					"certificate_authorities": certificates,
				},
			}
		}

	}

	// Elasticsearch output
	if fb.Spec.ElasticsearchRef.IsExternal() || fb.Spec.ElasticsearchRef.IsManaged() {
		certificates := make([]string, 0)

		// Compute Elasticsearch pki ca
		if es != nil && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
			certificates = append(certificates, "/usr/share/filebeat/es-ca/ca.crt")
		}

		// Compute external certificates
		if elasticsearchCASecret != nil {
			for certificateName := range elasticsearchCASecret.Data {
				if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
					certificates = append(certificates, fmt.Sprintf("/usr/share/filebeat/es-custom-ca/%s", certificateName))
				}
			}
		}
		if len(certificates) == 0 {
			filebeatConf["output.elasticsearch"] = map[string]any{
				"hosts":    elasticsearchHosts,
				"username": "${ELASTICSEARCH_USERNAME}",
				"password": "${ELASTICSEARCH_PASSWORD}",
			}
		} else {
			filebeatConf["output.elasticsearch"] = map[string]any{
				"hosts":    elasticsearchHosts,
				"username": "${ELASTICSEARCH_USERNAME}",
				"password": "${ELASTICSEARCH_PASSWORD}",
				"ssl": map[string]any{
					"enable":                  true,
					"certificate_authorities": certificates,
				},
			}
		}
	}

	injectedConfigMap := map[string]string{
		"filebeat.yml": helper.ToYamlOrDie(filebeatConf),
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
