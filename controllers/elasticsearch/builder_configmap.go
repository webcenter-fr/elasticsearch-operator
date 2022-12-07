package elasticsearch

import (
	"github.com/pkg/errors"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMaps permit to generate config maps for each node Groups
func BuildConfigMaps(es *elasticsearchapi.Elasticsearch) (configMaps []corev1.ConfigMap, err error) {
	var (
		configMap      corev1.ConfigMap
		expectedConfig map[string]string
	)

	configMaps = make([]corev1.ConfigMap, 0, len(es.Spec.NodeGroups))
	injectedConfigMap := map[string]string{
		"elasticsearch.yml": `
xpack.security.enabled: true
xpack.security.authc.realms.file.file1.order: -100
xpack.security.authc.realms.native.native1.order: -99
xpack.security.transport.ssl.enabled: true
xpack.security.transport.ssl.verification_mode: certificate
xpack.security.transport.ssl.certificate: /usr/share/elasticsearch/config/transport-cert/${POD_NAME}.crt
xpack.security.transport.ssl.key: /usr/share/elasticsearch/config/transport-cert/${POD_NAME}.key
xpack.security.transport.ssl.certificate_authorities: /usr/share/elasticsearch/config/transport-cert/ca.crt
`,
	}

	if es.IsTlsApiEnabled() {
		injectedConfigMap["elasticsearch.yml"] += `
xpack.security.http.ssl.enabled: true
xpack.security.http.ssl.certificate: /usr/share/elasticsearch/config/api-cert/tls.crt
xpack.security.http.ssl.key: /usr/share/elasticsearch/config/api-cert/tls.key
xpack.security.http.ssl.certificate_authorities: /usr/share/elasticsearch/config/api-cert/ca.crt
`
	} else {
		injectedConfigMap["elasticsearch.yml"] += "xpack.security.http.ssl.enabled: false\n"
	}

	for _, nodeGroup := range es.Spec.NodeGroups {

		if es.Spec.GlobalNodeGroup.Config != nil {
			expectedConfig, err = helper.MergeSettings(nodeGroup.Config, es.Spec.GlobalNodeGroup.Config)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when merge config from global config and node group config %s", nodeGroup.Name)
			}
		} else {
			expectedConfig = nodeGroup.Config
		}

		// Inject computed config
		expectedConfig, err = helper.MergeSettings(injectedConfigMap, expectedConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when merge expected config with computed config on node group %s", nodeGroup.Name)
		}

		configMap = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetNodeGroupConfigMapName(es, nodeGroup.Name),
				Labels:      getLabels(es, map[string]string{"nodeGroup": nodeGroup.Name}),
				Annotations: getAnnotations(es),
			},
			Data: expectedConfig,
		}
		configMaps = append(configMaps, configMap)
	}

	return configMaps, nil
}
