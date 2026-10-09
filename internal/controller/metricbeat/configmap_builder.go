package metricbeat

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/discover"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildConfigMap permit to generate config maps
func buildConfigMaps(mb *beatcrd.Metricbeat, es *elasticsearchcrd.Elasticsearch, elasticsearchCASecret *corev1.Secret, discoverOutputType discovercrd.DiscoverType, discoverOutputSecretEnv *corev1.Secret, discoverOutputSecretFile *corev1.Secret) (configMaps []*corev1.ConfigMap, err error) {
	configMaps = make([]*corev1.ConfigMap, 0, 1)
	var cm *corev1.ConfigMap

	// ConfigMap that store configs
	var expectedConfig map[string]string

	// Compute the Elasticsearch hosts
	elasticsearchHosts := make([]string, 0, 1)
	if es != nil {
		elasticsearchHosts = append(elasticsearchHosts, elasticsearchcontrollers.GetPublicUrl(es, mb.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup, false))
	} else if mb.Spec.ElasticsearchRef.IsExternal() && len(mb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses) > 0 {
		elasticsearchHosts = mb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses
	}

	// Static config
	metricbeatConf := map[string]any{
		"http.enabled": true,
		"http.host":    "0.0.0.0",
		"metricbeat.config.modules": map[string]any{
			"path": "${path.config}/modules.d/*.yml",
		},
	}

	// Compute discover output
	if discoverOutputType == discovercrd.DiscoverTypeElasticsearch {
		// Get the environment variables for elasticsearch hosts
		elasticsearchHostsEnvVar := ""
		elasticsearchUsernameEnvVar := ""
		elasticsearchPasswordEnvVar := ""
		for key := range discoverOutputSecretEnv.Data {
			if strings.HasPrefix(key, "ELASTICSEARCH_HOSTS_") {
				elasticsearchHostsEnvVar = key
			} else if strings.HasPrefix(key, "ELASTICSEARCH_USERNAME_") {
				elasticsearchUsernameEnvVar = key
			} else if strings.HasPrefix(key, "ELASTICSEARCH_PASSWORD_") {
				elasticsearchPasswordEnvVar = key
			}
		}
		if elasticsearchHostsEnvVar == "" {
			return nil, errors.New("Cannot find elasticsearch hosts environment variable in discover output secret")
		}

		output := map[string]any{
			"hosts": []string{fmt.Sprintf("${%s}", elasticsearchHostsEnvVar)},
		}
		if elasticsearchUsernameEnvVar != "" && elasticsearchPasswordEnvVar != "" {
			output["username"] = fmt.Sprintf("${%s}", elasticsearchUsernameEnvVar)
			output["password"] = fmt.Sprintf("${%s}", elasticsearchPasswordEnvVar)
		}

		// Check if tls is enabled
		if discoverOutputSecretFile.Data["ca.crt"] != nil {
			output["ssl"] = map[string]any{
				"enabled":                 true,
				"certificate_authorities": []string{fmt.Sprintf("/usr/share/metricbeat/discover/%s/ca.crt", discover.GetDiscoverMountPathFromAnnotations(discoverOutputSecretFile))},
			}
		}

		metricbeatConf["output.elasticsearch"] = output
	}

	// Elasticsearch output
	if mb.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() || mb.Spec.ElasticsearchRef.IsExternal() {
		certificates := make([]string, 0)

		// Compute Elasticsearch pki ca
		if es != nil && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
			certificates = append(certificates, "/usr/share/metricbeat/es-ca/ca.crt")
		}

		// Compute external certificates
		if elasticsearchCASecret != nil {
			for certificateName := range elasticsearchCASecret.Data {
				if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
					certificates = append(certificates, fmt.Sprintf("/usr/share/metricbeat/es-custom-ca/%s", certificateName))
				}
			}
		}
		if len(certificates) == 0 {
			metricbeatConf["output.elasticsearch"] = map[string]any{
				"hosts":    elasticsearchHosts,
				"username": "${ELASTICSEARCH_USERNAME}",
				"password": "${ELASTICSEARCH_PASSWORD}",
			}
		} else {
			metricbeatConf["output.elasticsearch"] = map[string]any{
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
		"metricbeat.yml": helper.ToYamlOrDie(metricbeatConf),
	}
	config, err := yaml.Marshal(mb.Spec.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when unmarshall config")
	}

	configs := map[string]string{
		"metricbeat.yml": string(config),
	}
	if mb.Spec.ExtraConfigs != nil {
		configs, err = helper.MergeSettings(configs, mb.Spec.ExtraConfigs)
		if err != nil {
			return nil, errors.Wrap(err, "Error when merge config and extra configs")
		}
	}

	// Inject computed config
	expectedConfig, err = helper.MergeSettings(configs, injectedConfigMap)
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

	configMaps = append(configMaps, cm)

	// ConfigMap that store modules
	if mb.Spec.Modules != nil && len(mb.Spec.Modules.Data) > 0 {
		modules := map[string]string{}
		for module, data := range mb.Spec.Modules.Data {
			b, err := yaml.Marshal(data)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when marshall module %s", module)
			}
			modules[module] = string(b)
		}
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   mb.Namespace,
				Name:        GetConfigMapModuleName(mb),
				Labels:      getLabels(mb),
				Annotations: getAnnotations(mb),
			},
			Data: modules,
		}

		configMaps = append(configMaps, cm)
	}

	return configMaps, nil
}
