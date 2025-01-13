package elasticsearch

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildConfigMaps permit to generate config maps for each node Groups
func buildConfigMaps(es *elasticsearchcrd.Elasticsearch) (configMaps []corev1.ConfigMap, err error) {
	var (
		configMap      corev1.ConfigMap
		expectedConfig map[string]string
	)

	configMaps = make([]corev1.ConfigMap, 0, len(es.Spec.NodeGroups)+1)

	// Compute configmap that store Elasticsearch settings
	elasticsearchConfig := map[string]any{
		"xpack.security.enabled":                               true,
		"xpack.security.authc.realms.file.file1.order":         -100,
		"xpack.security.authc.realms.native.native1.order":     -99,
		"xpack.security.transport.ssl.enabled":                 true,
		"xpack.security.transport.ssl.verification_mode":       "full",
		"xpack.security.transport.ssl.certificate":             "/usr/share/elasticsearch/config/transport-cert/${POD_NAME}.crt",
		"xpack.security.transport.ssl.key":                     "/usr/share/elasticsearch/config/transport-cert/${POD_NAME}.key",
		"xpack.security.transport.ssl.certificate_authorities": "/usr/share/elasticsearch/config/transport-cert/ca.crt",
	}

	if es.Spec.Tls.IsTlsEnabled() {
		elasticsearchConfig["xpack.security.http.ssl.enabled"] = true
		elasticsearchConfig["xpack.security.http.ssl.certificate"] = "/usr/share/elasticsearch/config/api-cert/tls.crt"
		elasticsearchConfig["xpack.security.http.ssl.key"] = "/usr/share/elasticsearch/config/api-cert/tls.key"
		elasticsearchConfig["xpack.security.http.ssl.certificate_authorities"] = "/usr/share/elasticsearch/config/api-cert/ca.crt"
	} else {
		elasticsearchConfig["xpack.security.http.ssl.enabled"] = false
	}

	injectedConfigMap := map[string]string{
		"elasticsearch.yml": helper.ToYamlOrDie(elasticsearchConfig),
	}
	globalConfig, err := yaml.Marshal(es.Spec.GlobalNodeGroup.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when unmarshall global config")
	}
	globalConfigs := map[string]string{
		"elasticsearch.yml": string(globalConfig),
	}
	if es.Spec.GlobalNodeGroup.ExtraConfigs != nil {
		globalConfigs, err = helper.MergeSettings(globalConfigs, es.Spec.GlobalNodeGroup.ExtraConfigs)
		if err != nil {
			return nil, errors.Wrap(err, "Error when merge global config and global extra configs")
		}
	}

	for _, nodeGroup := range es.Spec.NodeGroups {
		config, err := yaml.Marshal(nodeGroup.Config)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when unmarshall config from node group %s", nodeGroup.Name)
		}
		configs := map[string]string{
			"elasticsearch.yml": string(config),
		}
		if nodeGroup.ExtraConfigs != nil {
			configs, err = helper.MergeSettings(configs, nodeGroup.ExtraConfigs)
			if err != nil {
				return nil, errors.Wrap(err, "Error when merge global config and global extra configs")
			}
		}

		expectedConfig, err = helper.MergeSettings(configs, globalConfigs)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when merge config from global config and node group config %s", nodeGroup.Name)
			}

		// Inject computed config
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, expectedConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when merge expected config with computed config on node group %s", nodeGroup.Name)
		}

		configMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: es.Namespace,
				Name:      GetNodeGroupConfigMapName(es, nodeGroup.Name),
				Labels:    getLabels(es, map[string]string{"nodeGroup": nodeGroup.Name}),
				Annotations: getAnnotations(es, map[string]string{
					fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey): "config",
				}),
			},
			Data: expectedConfig,
		}
		configMaps = append(configMaps, configMap)
	}

	// Compute configmap that store the bootstrapping properties
	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: es.Namespace,
			Name:      GetBootstrappingConfigMapName(es),
			Labels:    getLabels(es),
			Annotations: getAnnotations(es, map[string]string{
				fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey): "bootstrapping",
			}),
		},
		Data: make(map[string]string),
	}

	if len(es.Spec.NodeGroups) == 1 && es.Spec.NodeGroups[0].Replicas == 1 {
		// Cluster with only one node
		configMap.Data["discovery.type"] = "single-node"
	} else {
		if !es.IsBoostrapping() {
			// Cluster with multiple nodes
			configMap.Data["cluster.initial_master_nodes"] = computeInitialMasterNodes(es)
		}

		configMap.Data["discovery.seed_hosts"] = computeDiscoverySeedHosts(es)
	}

	configMaps = append(configMaps, configMap)

	return configMaps, nil
}

// computeInitialMasterNodes create the list of all master nodes
func computeInitialMasterNodes(es *elasticsearchcrd.Elasticsearch) string {
	masterNodes := make([]string, 0, 3)
	for _, nodeGroup := range es.Spec.NodeGroups {
		if IsMasterRole(es, nodeGroup.Name) {
			masterNodes = append(masterNodes, GetNodeGroupNodeNames(es, nodeGroup.Name)...)
		}
	}

	return strings.Join(masterNodes, ", ")
}

// computeDiscoverySeedHosts create the list of all headless service of all masters node groups
func computeDiscoverySeedHosts(es *elasticsearchcrd.Elasticsearch) string {
	serviceNames := make([]string, 0, 1)

	for _, nodeGroup := range es.Spec.NodeGroups {
		if IsMasterRole(es, nodeGroup.Name) {
			serviceNames = append(serviceNames, GetNodeGroupServiceNameHeadless(es, nodeGroup.Name))
		}
	}

	return strings.Join(serviceNames, ", ")
}
